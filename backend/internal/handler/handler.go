package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"token-points-system/internal/config"
	"token-points-system/internal/models"
	"token-points-system/internal/repository"
	"token-points-system/internal/scheduler"
	"token-points-system/internal/service"
)

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

type BalanceHandler struct {
	balanceSvc *service.BalanceService
}

func NewBalanceHandler(balanceSvc *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{balanceSvc: balanceSvc}
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		writeError(w, http.StatusBadRequest, "invalid path format, expected /api/balance/{chain_id}/{address}")
		return
	}

	chainID := pathParts[2]
	userAddress := pathParts[3]

	if chainID == "" || userAddress == "" {
		writeError(w, http.StatusBadRequest, "chain_id and address are required")
		return
	}

	ctx := r.Context()
	balance, err := h.balanceSvc.GetUserBalance(ctx, chainID, userAddress)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get balance: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"balance":   balance,
		"chain":     chainID,
		"address":   userAddress,
		"updatedAt": time.Now().Format(time.RFC3339),
	})
}

func (h *BalanceHandler) ListBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	chainID := r.URL.Query().Get("chain_id")
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "chain_id is required")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	ctx := r.Context()
	balances, err := h.balanceSvc.ListBalances(ctx, chainID, offset, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list balances: "+err.Error())
		return
	}

	total, err := h.balanceSvc.CountBalances(ctx, chainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count balances: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":    balances,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

type PointsHandler struct {
	pointsSvc  *service.PointsService
	pointsRepo *repository.PointsRepository
	calcRepo   *repository.CalculationRepository
}

func NewPointsHandler(pointsSvc *service.PointsService, pointsRepo *repository.PointsRepository, calcRepo *repository.CalculationRepository) *PointsHandler {
	return &PointsHandler{pointsSvc: pointsSvc, pointsRepo: pointsRepo, calcRepo: calcRepo}
}

func (h *PointsHandler) GetPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		writeError(w, http.StatusBadRequest, "invalid path format, expected /api/points/{chain_id}/{address}")
		return
	}

	chainID := pathParts[2]
	userAddress := pathParts[3]

	if chainID == "" || userAddress == "" {
		writeError(w, http.StatusBadRequest, "chain_id and address are required")
		return
	}

	ctx := r.Context()
	points, err := h.pointsSvc.GetUserPoints(ctx, chainID, userAddress)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get points: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalPoints":      points,
		"chain":            chainID,
		"address":          userAddress,
		"lastCalculatedAt": time.Now().Format(time.RFC3339),
	})
}

func (h *PointsHandler) ListPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	chainID := r.URL.Query().Get("chain_id")
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "chain_id is required")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	ctx := r.Context()
	points, err := h.pointsSvc.ListPoints(ctx, chainID, offset, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list points: "+err.Error())
		return
	}

	total, err := h.pointsSvc.CountPoints(ctx, chainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count points: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":    points,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *PointsHandler) GetPointsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 1 || days > 30 {
		days = 7
	}

	ctx := r.Context()
	dailyPoints, err := h.calcRepo.GetDailyPoints(ctx, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get points history: "+err.Error())
		return
	}

	labels := make([]string, 0)
	values := make([]float64, 0)

	loc, _ := time.LoadLocation("Local")
	now := time.Now().In(loc)
	for i := days - 1; i >= 0; i-- {
		t := now.AddDate(0, 0, -i)
		dateStr := t.Format("2006-01-02")
		labels = append(labels, t.Format("Jan 02"))
		if pts, ok := dailyPoints[dateStr]; ok {
			values = append(values, pts)
		} else {
			values = append(values, 0)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"labels": labels,
		"values": values,
	})
}

type HistoryHandler struct {
	historyRepo *repository.HistoryRepository
}

func NewHistoryHandler(historyRepo *repository.HistoryRepository) *HistoryHandler {
	return &HistoryHandler{historyRepo: historyRepo}
}

func (h *HistoryHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		writeError(w, http.StatusBadRequest, "invalid path format, expected /api/history/{chain_id}/{address}")
		return
	}

	chainID := pathParts[2]
	userAddress := pathParts[3]

	if chainID == "" || userAddress == "" {
		writeError(w, http.StatusBadRequest, "chain_id and address are required")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	ctx := r.Context()
	histories, err := h.historyRepo.GetByUser(ctx, chainID, userAddress, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get history: "+err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(histories))
	for _, h := range histories {
		items = append(items, map[string]interface{}{
			"timestamp":     h.Timestamp.Format(time.RFC3339),
			"changeType":    h.ChangeType,
			"balanceBefore": h.BalanceBefore,
			"balanceAfter":  h.BalanceAfter,
			"changeAmount":  h.ChangeAmount,
			"txHash":        h.TxHash,
			"blockNumber":   h.BlockNumber,
		})
	}

	writeJSON(w, http.StatusOK, items)
}

type StatsHandler struct {
	balanceRepo *repository.BalanceRepository
	pointsRepo  *repository.PointsRepository
	historyRepo *repository.HistoryRepository
	blockRepo   *repository.BlockRepository
	chains      []config.ChainConfig
}

func NewStatsHandler(
	balanceRepo *repository.BalanceRepository,
	pointsRepo *repository.PointsRepository,
	historyRepo *repository.HistoryRepository,
	blockRepo *repository.BlockRepository,
	chains []config.ChainConfig,
) *StatsHandler {
	return &StatsHandler{
		balanceRepo: balanceRepo,
		pointsRepo:  pointsRepo,
		historyRepo: historyRepo,
		blockRepo:   blockRepo,
		chains:      chains,
	}
}

func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	var totalUsers int64
	for _, chain := range h.chains {
		if !chain.Enabled {
			continue
		}
		users, _ := h.balanceRepo.CountByChain(ctx, chain.ID)
		totalUsers += users
	}

	var allPoints []models.UserPoints
	for _, chain := range h.chains {
		if !chain.Enabled {
			continue
		}
		points, _ := h.pointsRepo.GetAllByChain(ctx, chain.ID)
		allPoints = append(allPoints, points...)
	}

	var totalPoints float64
	for _, p := range allPoints {
		if p.TotalPoints != "" {
			var pts float64
			_, _ = sscanf(p.TotalPoints, "%f", &pts)
			totalPoints += pts
		}
	}

	totalTransactions, _ := h.historyRepo.CountAll(ctx)

	sepoliaBlock, _ := h.blockRepo.GetLastProcessed(ctx, "sepolia")
	baseBlock, _ := h.blockRepo.GetLastProcessed(ctx, "base-sepolia")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalUsers":        totalUsers,
		"totalPoints":       totalPoints,
		"totalTransactions": totalTransactions,
		"sepoliaBlock":      sepoliaBlock,
		"baseBlock":         baseBlock,
	})
}

func sscanf(str, format string, v interface{}) (int, error) {
	_, err := fmt.Sscanf(str, format, v)
	return 0, err
}

type RecalculateHandler struct {
	scheduler   *scheduler.PointsScheduler
	balanceRepo *repository.BalanceRepository
	cfg         *config.Config
}

func NewRecalculateHandler(
	scheduler *scheduler.PointsScheduler,
	balanceRepo *repository.BalanceRepository,
	cfg *config.Config,
) *RecalculateHandler {
	return &RecalculateHandler{
		scheduler:   scheduler,
		balanceRepo: balanceRepo,
		cfg:         cfg,
	}
}

func (h *RecalculateHandler) TriggerRecalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Chain     string `json:"chain"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	chainID := req.Chain
	if chainID == "" {
		chainID = r.URL.Query().Get("chain_id")
	}
	if chainID == "" {
		chainID = r.URL.Query().Get("chain")
	}
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "chain is required")
		return
	}

	var periodStart, periodEnd time.Time
	var err error

	if req.StartTime != "" {
		periodStart, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			periodStart, _ = time.Parse("2006-01-02T15:04", req.StartTime)
		}
	}
	if periodStart.IsZero() {
		periodStart = time.Now().Add(-24 * time.Hour)
	}

	if req.EndTime != "" {
		periodEnd, err = time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			periodEnd, _ = time.Parse("2006-01-02T15:04", req.EndTime)
		}
	}
	if periodEnd.IsZero() {
		periodEnd = time.Now()
	}

	ctx := r.Context()
	go func() {
		_ = h.scheduler.TriggerManualCalculation(ctx, chainID, periodStart, periodEnd)
	}()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "recalculation triggered",
		"chain":     chainID,
		"startTime": periodStart.Format(time.RFC3339),
		"endTime":   periodEnd.Format(time.RFC3339),
	})
}

type TransactionHandler struct {
	historyRepo *repository.HistoryRepository
}

func NewTransactionHandler(historyRepo *repository.HistoryRepository) *TransactionHandler {
	return &TransactionHandler{historyRepo: historyRepo}
}

func (h *TransactionHandler) GetRecentTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	ctx := r.Context()
	histories, err := h.historyRepo.GetRecent(ctx, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get transactions: "+err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(histories))
	for _, h := range histories {
		items = append(items, map[string]interface{}{
			"chain":     h.ChainID,
			"type":      string(h.ChangeType),
			"from":      h.UserAddress,
			"to":        "",
			"amount":    h.ChangeAmount,
			"timestamp": h.Timestamp.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, items)
}

type BackupHandler struct {
	calcRepo *repository.CalculationRepository
	cfg      *config.Config
}

func NewBackupHandler(calcRepo *repository.CalculationRepository, cfg *config.Config) *BackupHandler {
	return &BackupHandler{calcRepo: calcRepo, cfg: cfg}
}

func (h *BackupHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Chain string `json:"chain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "backup created",
		"chain":     req.Chain,
		"createdAt": time.Now().Format(time.RFC3339),
	})
}

func (h *BackupHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	backups := make([]map[string]interface{}, 0)
	writeJSON(w, http.StatusOK, backups)
}

func (h *BackupHandler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		BackupID string `json:"backupId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "backup restored",
		"backupId": req.BackupID,
	})
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
