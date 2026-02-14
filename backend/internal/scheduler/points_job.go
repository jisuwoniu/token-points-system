package scheduler

import (
	"context"
	"time"

	"token-points-system/internal/config"
	"token-points-system/internal/repository"
	"token-points-system/internal/service"
	"token-points-system/pkg/logger"

	"github.com/robfig/cron/v3"
)

type PointsScheduler struct {
	cron        *cron.Cron
	pointsSvc   *service.PointsService
	balanceRepo *repository.BalanceRepository
	chains      []config.ChainConfig
	cronExpr    string
}

// NewPointsScheduler 创建积分调度器
func NewPointsScheduler(
	pointsSvc *service.PointsService,
	balanceRepo *repository.BalanceRepository,
	chains []config.ChainConfig,
	cronExpr string,
) *PointsScheduler {
	if cronExpr == "" {
		cronExpr = "0 0 * * * *"
	}

	return &PointsScheduler{
		cron:        cron.New(cron.WithSeconds()),
		pointsSvc:   pointsSvc,
		balanceRepo: balanceRepo,
		chains:      chains,
		cronExpr:    cronExpr,
	}
}

// Start 启动积分计算调度器
func (s *PointsScheduler) Start() error {
	_, err := s.cron.AddFunc(s.cronExpr, s.calculatePoints)
	if err != nil {
		return err
	}

	s.cron.Start()
	logger.Info("积分计算调度器已启动")
	return nil
}

// Stop 停止积分计算调度器
func (s *PointsScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	logger.Info("积分计算调度器已停止")
}

// calculatePoints 执行积分计算
func (s *PointsScheduler) calculatePoints() {
	ctx := context.Background()
	now := time.Now()
	periodEnd := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	periodStart := periodEnd.Add(-time.Hour)

	logger.WithFields(map[string]interface{}{
		"period_start": periodStart,
		"period_end":   periodEnd,
	}).Info("开始积分计算")

	for _, chain := range s.chains {
		if !chain.Enabled {
			continue
		}

		go s.calculatePointsForChain(ctx, chain.ID, periodStart, periodEnd)
	}
}

// calculatePointsForChain 为指定链计算积分
func (s *PointsScheduler) calculatePointsForChain(ctx context.Context, chainID string, periodStart, periodEnd time.Time) {
	pageSize := 100
	offset := 0

	totalUsers, err := s.balanceRepo.CountByChain(ctx, chainID)
	if err != nil {
		logger.Error("获取链用户数量失败:", chainID, err)
		return
	}

	logger.WithFields(map[string]interface{}{
		"chain_id":    chainID,
		"total_users": totalUsers,
	}).Info("分页处理链用户")

	processedCount := 0

	for {
		balances, err := s.balanceRepo.GetByChainPaginated(ctx, chainID, offset, pageSize)
		if err != nil {
			logger.Error("获取链余额数据失败:", chainID, err)
			break
		}

		if len(balances) == 0 {
			break
		}

		for _, balance := range balances {
			_, err := s.pointsSvc.CalculatePointsForUser(ctx, chainID, balance.UserAddress, periodStart, periodEnd)
			if err != nil {
				logger.Error("计算用户积分失败:", balance.UserAddress, err)
				continue
			}
			processedCount++
		}

		offset += pageSize

		if len(balances) < pageSize {
			break
		}
	}

	logger.WithFields(map[string]interface{}{
		"chain_id":        chainID,
		"processed_count": processedCount,
	}).Info("链积分计算完成")
}

// TriggerManualCalculation 手动触发积分计算
func (s *PointsScheduler) TriggerManualCalculation(ctx context.Context, chainID string, periodStart, periodEnd time.Time) error {
	pageSize := 100
	offset := 0

	for {
		balances, err := s.balanceRepo.GetByChainPaginated(ctx, chainID, offset, pageSize)
		if err != nil {
			return err
		}

		if len(balances) == 0 {
			break
		}

		for _, balance := range balances {
			_, err := s.pointsSvc.CalculatePointsForUser(ctx, chainID, balance.UserAddress, periodStart, periodEnd)
			if err != nil {
				logger.Error("计算用户积分失败:", balance.UserAddress, err)
			}
		}

		offset += pageSize

		if len(balances) < pageSize {
			break
		}
	}

	return nil
}
