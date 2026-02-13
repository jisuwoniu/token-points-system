package scheduler

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"token-points-system/internal/config"
	"token-points-system/internal/repository"
	"token-points-system/internal/service"
	"token-points-system/pkg/logger"
)

type PointsScheduler struct {
	cron        *cron.Cron
	pointsSvc   *service.PointsService
	balanceRepo *repository.BalanceRepository
	chains      []config.ChainConfig
}

func NewPointsScheduler(
	pointsSvc *service.PointsService,
	balanceRepo *repository.BalanceRepository,
	chains []config.ChainConfig,
	cronExpr string,
) *PointsScheduler {
	return &PointsScheduler{
		cron:        cron.New(cron.WithSeconds()),
		pointsSvc:   pointsSvc,
		balanceRepo: balanceRepo,
		chains:      chains,
	}
}

func (s *PointsScheduler) Start() error {
	_, err := s.cron.AddFunc("0 0 * * * *", s.calculatePoints)
	if err != nil {
		return err
	}
	
	s.cron.Start()
	logger.Info("Points calculation scheduler started")
	return nil
}

func (s *PointsScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	logger.Info("Points calculation scheduler stopped")
}

func (s *PointsScheduler) calculatePoints() {
	ctx := context.Background()
	now := time.Now()
	periodEnd := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	periodStart := periodEnd.Add(-time.Hour)
	
	logger.WithFields(map[string]interface{}{
		"period_start": periodStart,
		"period_end":   periodEnd,
	}).Info("Starting points calculation")
	
	for _, chain := range s.chains {
		if !chain.Enabled {
			continue
		}
		
		go s.calculatePointsForChain(ctx, chain.ID, periodStart, periodEnd)
	}
}

func (s *PointsScheduler) calculatePointsForChain(ctx context.Context, chainID string, periodStart, periodEnd time.Time) {
	balances, err := s.balanceRepo.GetAllByChain(ctx, chainID)
	if err != nil {
		logger.Error("Failed to get balances for chain:", chainID, err)
		return
	}
	
	logger.WithFields(map[string]interface{}{
		"chain_id":     chainID,
		"users_count":  len(balances),
	}).Info("Processing chain users")
	
	for _, balance := range balances {
		_, err := s.pointsSvc.CalculatePointsForUser(ctx, chainID, balance.UserAddress, periodStart, periodEnd)
		if err != nil {
			logger.Error("Failed to calculate points for user:", balance.UserAddress, err)
			continue
		}
	}
	
	logger.WithFields(map[string]interface{}{
		"chain_id": chainID,
	}).Info("Points calculation completed for chain")
}

func (s *PointsScheduler) TriggerManualCalculation(ctx context.Context, chainID string, periodStart, periodEnd time.Time) error {
	balances, err := s.balanceRepo.GetAllByChain(ctx, chainID)
	if err != nil {
		return err
	}
	
	for _, balance := range balances {
		_, err := s.pointsSvc.CalculatePointsForUser(ctx, chainID, balance.UserAddress, periodStart, periodEnd)
		if err != nil {
			logger.Error("Failed to calculate points for user:", balance.UserAddress, err)
		}
	}
	
	return nil
}
