package repository

import (
	"context"
	"time"
	
	"gorm.io/gorm"
	"token-points-system/internal/models"
)

type HistoryRepository struct {
	db *gorm.DB
}

func NewHistoryRepository(db *gorm.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) Create(ctx context.Context, history *models.BalanceHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *HistoryRepository) GetByUser(ctx context.Context, chainID, userAddress string, limit int) ([]models.BalanceHistory, error) {
	var histories []models.BalanceHistory
	query := r.db.WithContext(ctx).
		Where("chain_id = ? AND user_address = ?", chainID, userAddress).
		Order("timestamp DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&histories).Error
	return histories, err
}

func (r *HistoryRepository) GetByTimeRange(ctx context.Context, chainID string, start, end time.Time) ([]models.BalanceHistory, error) {
	var histories []models.BalanceHistory
	err := r.db.WithContext(ctx).
		Where("chain_id = ? AND timestamp >= ? AND timestamp < ?", chainID, start, end).
		Order("timestamp ASC").
		Find(&histories).Error
	return histories, err
}

func (r *HistoryRepository) GetUserHistoryInRange(ctx context.Context, chainID, userAddress string, start, end time.Time) ([]models.BalanceHistory, error) {
	var histories []models.BalanceHistory
	err := r.db.WithContext(ctx).
		Where("chain_id = ? AND user_address = ? AND timestamp >= ? AND timestamp < ?", 
			chainID, userAddress, start, end).
		Order("timestamp ASC").
		Find(&histories).Error
	return histories, err
}

func (r *HistoryRepository) ExistsByTxHash(ctx context.Context, txHash string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.BalanceHistory{}).
		Where("tx_hash = ?", txHash).
		Count(&count).Error
	return count > 0, err
}
