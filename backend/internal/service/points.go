package service

import (
	"context"
	"math/big"
	"time"

	"token-points-system/internal/config"
	"token-points-system/internal/models"
	"token-points-system/internal/repository"
	"token-points-system/pkg/errors"
	"token-points-system/pkg/logger"
)

type PointsService struct {
	pointsRepo      *repository.PointsRepository
	historyRepo     *repository.HistoryRepository
	calcRepo        *repository.CalculationRepository
	calculationRate float64
}

func NewPointsService(
	pointsRepo *repository.PointsRepository,
	historyRepo *repository.HistoryRepository,
	calcRepo *repository.CalculationRepository,
	cfg *config.PointsConfig,
) *PointsService {
	return &PointsService{
		pointsRepo:      pointsRepo,
		historyRepo:     historyRepo,
		calcRepo:        calcRepo,
		calculationRate: cfg.CalculationRate,
	}
}

// CalculatePointsForUser 计算并发放用户在指定时间段的积分
// 基于余额历史进行分钟级精确计算
// 使用计算哈希确保幂等性，防止重复计算
func (s *PointsService) CalculatePointsForUser(ctx context.Context, chainID, userAddress string, periodStart, periodEnd time.Time) (string, error) {
	hash := s.calcRepo.GenerateHash(chainID, userAddress, periodStart, periodEnd)

	exists, err := s.calcRepo.ExistsByHash(ctx, hash)
	if err != nil {
		return "0", errors.New(errors.ErrPointsCalc, "检查计算是否存在失败", err)
	}
	if exists {
		logger.WithFields(map[string]interface{}{
			"hash": hash,
		}).Debug("计算已存在")
		return "0", nil
	}

	histories, err := s.historyRepo.GetUserHistoryInRange(ctx, chainID, userAddress, periodStart, periodEnd)
	if err != nil {
		return "0", errors.New(errors.ErrPointsCalc, "获取历史记录失败", err)
	}

	totalPoints := s.calculatePointsFromHistory(histories, periodStart, periodEnd)

	calc := &models.PointCalculation{
		ChainID:         chainID,
		UserAddress:     userAddress,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		PointsEarned:    totalPoints.String(),
		CalculationHash: hash,
	}

	if err := s.calcRepo.Create(ctx, calc); err != nil {
		return "0", errors.New(errors.ErrPointsCalc, "保存计算记录失败", err)
	}

	if err := s.pointsRepo.AddPoints(ctx, chainID, userAddress, totalPoints.String()); err != nil {
		return "0", errors.New(errors.ErrPointsCalc, "更新总积分失败", err)
	}

	logger.WithFields(map[string]interface{}{
		"chain_id":      chainID,
		"user_address":  userAddress,
		"points_earned": totalPoints.String(),
		"period_start":  periodStart,
		"period_end":    periodEnd,
	}).Info("积分已计算")

	return totalPoints.String(), nil
}

// calculatePointsFromHistory 基于余额历史计算积分
// 公式：积分 = 余额 × 费率 × 持有时长（小时）
// 时长精确到分钟级别
func (s *PointsService) calculatePointsFromHistory(histories []models.BalanceHistory, periodStart, periodEnd time.Time) *big.Float {
	if len(histories) == 0 {
		return big.NewFloat(0)
	}

	totalPoints := big.NewFloat(0)
	rate := big.NewFloat(s.calculationRate)

	var currentBalance *big.Int
	var currentStart time.Time

	for i, h := range histories {
		if i == 0 {
			currentBalance, _ = new(big.Int).SetString(h.BalanceAfter, 10)
			currentStart = h.Timestamp
			continue
		}

		duration := h.Timestamp.Sub(currentStart).Minutes()
		balanceFloat := new(big.Float).SetInt(currentBalance)
		durationFloat := big.NewFloat(duration / 60.0)

		points := new(big.Float).Mul(balanceFloat, rate)
		points.Mul(points, durationFloat)
		totalPoints.Add(totalPoints, points)

		currentBalance, _ = new(big.Int).SetString(h.BalanceAfter, 10)
		currentStart = h.Timestamp
	}

	if currentStart.Before(periodEnd) {
		duration := periodEnd.Sub(currentStart).Minutes()
		balanceFloat := new(big.Float).SetInt(currentBalance)
		durationFloat := big.NewFloat(duration / 60.0)

		points := new(big.Float).Mul(balanceFloat, rate)
		points.Mul(points, durationFloat)
		totalPoints.Add(totalPoints, points)
	}

	return totalPoints
}

// GetUserPoints 获取用户在指定链上的总积分
func (s *PointsService) GetUserPoints(ctx context.Context, chainID, userAddress string) (string, error) {
	points, err := s.pointsRepo.GetByUser(ctx, chainID, userAddress)
	if err != nil {
		return "0", err
	}
	if points == nil {
		return "0", nil
	}
	return points.TotalPoints, nil
}

func (s *PointsService) ListPoints(ctx context.Context, chainID string, offset, limit int) ([]models.UserPoints, error) {
	return s.pointsRepo.GetByChainPaginated(ctx, chainID, offset, limit)
}

func (s *PointsService) CountPoints(ctx context.Context, chainID string) (int64, error) {
	return s.pointsRepo.CountByChain(ctx, chainID)
}
