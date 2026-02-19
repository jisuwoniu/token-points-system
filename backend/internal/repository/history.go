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

func (r *HistoryRepository) GetRecent(ctx context.Context, limit int) ([]models.BalanceHistory, error) {
	var histories []models.BalanceHistory
	if limit <= 0 {
		limit = 10
	}
	err := r.db.WithContext(ctx).
		Order("timestamp DESC").
		Limit(limit).
		Find(&histories).Error
	return histories, err
}

func (r *HistoryRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.BalanceHistory{}).
		Count(&count).Error
	return count, err
}

func (r *HistoryRepository) GetDailyTransactionCounts(ctx context.Context, days int) (map[string]int64, error) {
	type DailyCount struct {
		Date  string
		Count int64
	}

	var results []DailyCount
	startDate := time.Now().AddDate(0, 0, -days+1).Format("2006-01-02")

	err := r.db.WithContext(ctx).
		Model(&models.BalanceHistory{}).
		Select("DATE(timestamp) as date, COUNT(*) as count").
		Where("DATE(timestamp) >= ?", startDate).
		Group("DATE(timestamp)").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Date] = r.Count
	}
	return counts, nil
}
