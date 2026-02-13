package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"token-points-system/internal/blockchain"
	"token-points-system/internal/config"
	"token-points-system/internal/repository"
	"token-points-system/internal/scheduler"
	"token-points-system/internal/service"
	"token-points-system/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}
	
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	
	if err := logger.Init(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	
	db, err := initDatabase(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database:", err)
	}
	defer closeDatabase(db)
	
	balanceRepo := repository.NewBalanceRepository(db)
	pointsRepo := repository.NewPointsRepository(db)
	historyRepo := repository.NewHistoryRepository(db)
	blockRepo := repository.NewBlockRepository(db)
	calcRepo := repository.NewCalculationRepository(db)
	
	balanceSvc := service.NewBalanceService(balanceRepo, historyRepo, blockRepo)
	pointsSvc := service.NewPointsService(pointsRepo, historyRepo, calcRepo, &cfg.Points)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	for _, chainCfg := range cfg.GetEnabledChains() {
		go startChainListener(ctx, chainCfg, balanceSvc, blockRepo)
	}
	
	pointsScheduler := scheduler.NewPointsScheduler(pointsSvc, balanceRepo, cfg.Chains, cfg.Points.CalculationCron)
	if err := pointsScheduler.Start(); err != nil {
		logger.Fatal("Failed to start scheduler:", err)
	}
	defer pointsScheduler.Stop()
	
	router := setupHTTPRouter(balanceSvc, pointsSvc, pointsScheduler, cfg)
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}
	
	go func() {
		logger.Info("Server starting on port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed:", err)
		}
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Info("Shutting down server...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error:", err)
	}
	
	logger.Info("Server stopped")
}

func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	
	return db, nil
}

func closeDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to get database instance:", err)
		return
	}
	sqlDB.Close()
}

func startChainListener(ctx context.Context, chainCfg config.ChainConfig, balanceSvc *service.BalanceService, blockRepo *repository.BlockRepository) {
	client, err := blockchain.NewClient(&chainCfg)
	if err != nil {
		logger.Error("Failed to create blockchain client:", err)
		return
	}
	defer client.Close()
	
	listener := blockchain.NewEventListener(&chainCfg, client)
	
	go listener.Start(ctx, chainCfg.StartBlock)
	
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-listener.GetEventChannel():
			timestamp, err := client.GetBlockTimestamp(ctx, event.BlockNum)
			if err != nil {
				logger.Error("Failed to get block timestamp:", err)
				continue
			}
			
			if err := balanceSvc.ProcessTransfer(ctx, chainCfg.ID, event, timestamp); err != nil {
				logger.Error("Failed to process transfer:", err)
			}
		}
	}
}

func setupHTTPRouter(balanceSvc *service.BalanceService, pointsSvc *service.PointsService, scheduler *scheduler.PointsScheduler, cfg *config.Config) http.Handler {
	router := http.NewServeMux()
	
	router.HandleFunc("/api/balance/", handleGetBalance(balanceSvc))
	router.HandleFunc("/api/points/", handleGetPoints(pointsSvc))
	router.HandleFunc("/api/history/", handleGetHistory())
	router.HandleFunc("/api/stats", handleGetStats())
	router.HandleFunc("/api/recalculate", handleRecalculate(scheduler, cfg))
	router.HandleFunc("/api/backup", handleBackup())
	router.HandleFunc("/health", handleHealth)
	
	fs := http.FileServer(http.Dir("./web"))
	router.Handle("/", fs)
	
	return router
}

func handleGetBalance(balanceSvc *service.BalanceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

func handleGetPoints(pointsSvc *service.PointsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

func handleRecalculate(scheduler *scheduler.PointsScheduler, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func handleGetHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}
}

func handleGetStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"totalUsers":0,"totalPoints":0,"totalTransactions":0}`))
	}
}

func handleBackup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}
