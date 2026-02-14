package service

import (
	"context"
	"math/big"
	"sync"
	"time"

	"token-points-system/internal/blockchain"
	"token-points-system/internal/models"
	"token-points-system/internal/repository"
	"token-points-system/pkg/errors"
	"token-points-system/pkg/logger"
)

type BalanceService struct {
	balanceRepo *repository.BalanceRepository
	historyRepo *repository.HistoryRepository
	blockRepo   *repository.BlockRepository
	mu          sync.RWMutex
}

func NewBalanceService(
	balanceRepo *repository.BalanceRepository,
	historyRepo *repository.HistoryRepository,
	blockRepo *repository.BlockRepository,
) *BalanceService {
	return &BalanceService{
		balanceRepo: balanceRepo,
		historyRepo: historyRepo,
		blockRepo:   blockRepo,
	}
}

// ProcessTransfer 处理转账事件并更新用户余额
// 记录余额历史并通过交易哈希确保幂等性
func (s *BalanceService) ProcessTransfer(ctx context.Context, chainID string, event *blockchain.TransferEvent, timestamp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exists, err := s.historyRepo.ExistsByTxHash(ctx, event.TxHash)
	if err != nil {
		return errors.New(errors.ErrBalanceUpdate, "检查交易是否存在失败", err)
	}
	if exists {
		logger.WithFields(map[string]interface{}{
			"tx_hash": event.TxHash,
		}).Debug("交易已处理")
		return nil
	}

	if err := s.processUserTransfer(ctx, chainID, event.From.Hex(), event, timestamp); err != nil {
		return err
	}

	if event.From != event.To {
		if err := s.processUserTransfer(ctx, chainID, event.To.Hex(), event, timestamp); err != nil {
			return err
		}
	}

	return s.blockRepo.MarkProcessed(ctx, chainID, event.BlockNum)
}

// processUserTransfer 处理单个用户的转账
// 计算余额变动并记录到历史
func (s *BalanceService) processUserTransfer(ctx context.Context, chainID string, userAddr string, event *blockchain.TransferEvent, timestamp time.Time) error {
	currentBalance, err := s.balanceRepo.GetByUser(ctx, chainID, userAddr)
	if err != nil {
		return err
	}

	balanceBefore := big.NewInt(0)
	if currentBalance != nil {
		balanceBefore.SetString(currentBalance.Balance, 10)
	}

	changeAmount := event.GetChangeAmount(userAddr)
	balanceAfter := new(big.Int).Add(balanceBefore, changeAmount)

	if balanceAfter.Sign() < 0 {
		logger.WithFields(map[string]interface{}{
			"user_address":   userAddr,
			"balance_before": balanceBefore.String(),
			"change_amount":  changeAmount.String(),
		}).Error("检测到负余额")
		return errors.New(errors.ErrBalanceUpdate, "负余额", nil)
	}

	history := &models.BalanceHistory{
		ChainID:       chainID,
		UserAddress:   userAddr,
		BalanceBefore: balanceBefore.String(),
		BalanceAfter:  balanceAfter.String(),
		ChangeAmount:  changeAmount.String(),
		ChangeType:    event.DetermineChangeType(userAddr),
		TxHash:        event.TxHash,
		BlockNumber:   event.BlockNum,
		Timestamp:     timestamp,
	}

	if err := s.historyRepo.Create(ctx, history); err != nil {
		return errors.New(errors.ErrBalanceUpdate, "创建历史记录失败", err)
	}

	if err := s.balanceRepo.UpdateBalance(ctx, chainID, userAddr, balanceAfter.String()); err != nil {
		return errors.New(errors.ErrBalanceUpdate, "更新余额失败", err)
	}

	logger.WithFields(map[string]interface{}{
		"chain_id":       chainID,
		"user_address":   userAddr,
		"balance_before": balanceBefore.String(),
		"balance_after":  balanceAfter.String(),
		"change_type":    history.ChangeType,
	}).Info("余额已更新")

	return nil
}

// GetUserBalance 获取用户在指定链上的当前余额
func (s *BalanceService) GetUserBalance(ctx context.Context, chainID, userAddress string) (string, error) {
	balance, err := s.balanceRepo.GetByUser(ctx, chainID, userAddress)
	if err != nil {
		return "0", err
	}
	if balance == nil {
		return "0", nil
	}
	return balance.Balance, nil
}
