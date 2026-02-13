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

func (s *BalanceService) ProcessTransfer(ctx context.Context, chainID string, event *blockchain.TransferEvent, timestamp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	exists, err := s.historyRepo.ExistsByTxHash(ctx, event.TxHash)
	if err != nil {
		return errors.New(errors.ErrBalanceUpdate, "failed to check tx existence", err)
	}
	if exists {
		logger.WithFields(map[string]interface{}{
			"tx_hash": event.TxHash,
		}).Debug("Transaction already processed")
		return nil
	}
	
	if err := s.processUserTransfer(ctx, chainID, event.From.Hex(), event, timestamp, true); err != nil {
		return err
	}
	
	if event.From != event.To {
		if err := s.processUserTransfer(ctx, chainID, event.To.Hex(), event, timestamp, false); err != nil {
			return err
		}
	}
	
	return s.blockRepo.MarkProcessed(ctx, chainID, event.BlockNum)
}

func (s *BalanceService) processUserTransfer(ctx context.Context, chainID string, userAddr string, event *blockchain.TransferEvent, timestamp time.Time, isFrom bool) error {
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
			"user_address": userAddr,
			"balance_before": balanceBefore.String(),
			"change_amount": changeAmount.String(),
		}).Error("Negative balance detected")
		return errors.New(errors.ErrBalanceUpdate, "negative balance", nil)
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
		return errors.New(errors.ErrBalanceUpdate, "failed to create history", err)
	}
	
	if err := s.balanceRepo.UpdateBalance(ctx, chainID, userAddr, balanceAfter.String()); err != nil {
		return errors.New(errors.ErrBalanceUpdate, "failed to update balance", err)
	}
	
	logger.WithFields(map[string]interface{}{
		"chain_id":       chainID,
		"user_address":   userAddr,
		"balance_before": balanceBefore.String(),
		"balance_after":  balanceAfter.String(),
		"change_type":    history.ChangeType,
	}).Info("Balance updated")
	
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
