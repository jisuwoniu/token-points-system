package service

import (
	"context"
	"math/big"
	"strings"
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

const zeroAddress = "0x0000000000000000000000000000000000000000"

type BalanceClient interface {
	GetTokenBalance(ctx context.Context, userAddress string) (*big.Int, error)
}

func (s *BalanceService) ProcessTransfer(ctx context.Context, chainID string, event *blockchain.TransferEvent, timestamp time.Time, client BalanceClient) error {
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

	if strings.ToLower(event.From.Hex()) != zeroAddress {
		if err := s.processUserTransfer(ctx, chainID, event.From.Hex(), event, timestamp, client); err != nil {
			return err
		}
	}

	if event.From != event.To && strings.ToLower(event.To.Hex()) != zeroAddress {
		if err := s.processUserTransfer(ctx, chainID, event.To.Hex(), event, timestamp, client); err != nil {
			return err
		}
	}

	return s.blockRepo.MarkProcessed(ctx, chainID, event.BlockNum)
}

func (s *BalanceService) processUserTransfer(ctx context.Context, chainID string, userAddr string, event *blockchain.TransferEvent, timestamp time.Time, client BalanceClient) error {
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
		}).Warn("检测到负余额，尝试从链上同步")

		if client != nil {
			onchainBalance, err := client.GetTokenBalance(ctx, userAddr)
			if err != nil {
				logger.Error("从链上获取余额失败:", err)
				return errors.New(errors.ErrBalanceUpdate, "负余额且无法从链上同步", nil)
			}

			balanceBefore = onchainBalance
			balanceAfter = new(big.Int).Add(balanceBefore, changeAmount)

			if balanceAfter.Sign() < 0 {
				logger.WithFields(map[string]interface{}{
					"user_address":    userAddr,
					"onchain_balance": onchainBalance.String(),
					"change_amount":   changeAmount.String(),
					"balance_after":   balanceAfter.String(),
				}).Error("链上余额也不足")
				return errors.New(errors.ErrBalanceUpdate, "余额不足", nil)
			}

			logger.WithFields(map[string]interface{}{
				"user_address":    userAddr,
				"onchain_balance": onchainBalance.String(),
				"balance_after":   balanceAfter.String(),
			}).Info("从链上同步余额成功")
		} else {
			return errors.New(errors.ErrBalanceUpdate, "负余额", nil)
		}
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

func (s *BalanceService) ListBalances(ctx context.Context, chainID string, offset, limit int) ([]models.UserBalance, error) {
	return s.balanceRepo.GetByChainPaginated(ctx, chainID, offset, limit)
}

func (s *BalanceService) CountBalances(ctx context.Context, chainID string) (int64, error) {
	return s.balanceRepo.CountByChain(ctx, chainID)
}
