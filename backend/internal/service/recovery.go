package service

import (
	"context"
	"time"

	"token-points-system/internal/models"
	"token-points-system/internal/repository"
	"token-points-system/pkg/logger"
)

type RecoveryService struct {
	balanceRepo *repository.BalanceRepository
	pointsRepo  *repository.PointsRepository
	historyRepo *repository.HistoryRepository
	calcRepo    *repository.CalculationRepository
}

func NewRecoveryService(
	balanceRepo *repository.BalanceRepository,
	pointsRepo *repository.PointsRepository,
	historyRepo *repository.HistoryRepository,
	calcRepo *repository.CalculationRepository,
) *RecoveryService {
	return &RecoveryService{
		balanceRepo: balanceRepo,
		pointsRepo:  pointsRepo,
		historyRepo: historyRepo,
		calcRepo:    calcRepo,
	}
}

func (s *RecoveryService) BackupBalances(ctx context.Context, chainID string) error {
	balances, err := s.balanceRepo.GetAllByChain(ctx, chainID)
	if err != nil {
		return err
	}
	
	backupData := make(map[string]interface{})
	for _, b := range balances {
		backupData[b.UserAddress] = b.Balance
	}
	
	backup := &models.CalculationBackup{
		ChainID:    chainID,
		BackupType: models.BackupTypeBalanceSnapshot,
		BackupData: backupData,
	}
	
	return s.createBackup(ctx, backup)
}

func (s *RecoveryService) BackupPoints(ctx context.Context, chainID string) error {
	points, err := s.pointsRepo.GetAllByChain(ctx, chainID)
	if err != nil {
		return err
	}
	
	backupData := make(map[string]interface{})
	for _, p := range points {
		backupData[p.UserAddress] = map[string]interface{}{
			"total_points":        p.TotalPoints,
			"last_calculated_at":  p.LastCalculatedAt,
		}
	}
	
	backup := &models.CalculationBackup{
		ChainID:    chainID,
		BackupType: models.BackupTypePointsSnapshot,
		BackupData: backupData,
	}
	
	return s.createBackup(ctx, backup)
}

func (s *RecoveryService) RecalculateFromTime(ctx context.Context, chainID string, startTime time.Time, pointsSvc *PointsService) error {
	logger.WithFields(map[string]interface{}{
		"chain_id":    chainID,
		"start_time":  startTime,
	}).Info("Starting recalculation from time")
	
	balances, err := s.balanceRepo.GetAllByChain(ctx, chainID)
	if err != nil {
		return err
	}
	
	now := time.Now()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	
	startHour := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), startTime.Hour(), 0, 0, 0, startTime.Location())
	
	for _, balance := range balances {
		currentPeriod := startHour
		for currentPeriod.Before(currentHour) {
			nextPeriod := currentPeriod.Add(time.Hour)
			
			_, err := pointsSvc.CalculatePointsForUser(ctx, chainID, balance.UserAddress, currentPeriod, nextPeriod)
			if err != nil {
				logger.Error("Failed to recalculate points:", err)
			}
			
			currentPeriod = nextPeriod
		}
	}
	
	logger.WithFields(map[string]interface{}{
		"chain_id": chainID,
	}).Info("Recalculation completed")
	
	return nil
}

func (s *RecoveryService) createBackup(ctx context.Context, backup *models.CalculationBackup) error {
	return nil
}
