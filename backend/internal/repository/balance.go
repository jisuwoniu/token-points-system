package repository

import (
	"context"
	"errors"

	"token-points-system/internal/models"

	"gorm.io/gorm"
)

type BalanceRepository struct {
	db *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

// GetByUser 获取指定用户在指定链上的余额
func (r *BalanceRepository) GetByUser(ctx context.Context, chainID, userAddress string) (*models.UserBalance, error) {
	var balance models.UserBalance
	err := r.db.WithContext(ctx).
		Where("chain_id = ? AND user_address = ?", chainID, userAddress).
		First(&balance).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &balance, err
}

// UpdateBalance 更新或创建用户余额记录
func (r *BalanceRepository) UpdateBalance(ctx context.Context, chainID, userAddress, newBalance string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		balance := &models.UserBalance{
			ChainID:     chainID,
			UserAddress: userAddress,
			Balance:     newBalance,
		}

		result := tx.Where("chain_id = ? AND user_address = ?", chainID, userAddress).
			Assign(balance).
			FirstOrCreate(balance)

		return result.Error
	})
}

// GetAllByChain 获取指定链上所有用户的余额
// 警告：可能返回大量数据，生产环境请使用GetByChainPaginated
func (r *BalanceRepository) GetAllByChain(ctx context.Context, chainID string) ([]models.UserBalance, error) {
	var balances []models.UserBalance
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Find(&balances).Error
	return balances, err
}

// GetByChainPaginated 分页获取用户余额
// 生产环境使用此方法避免大数据集导致的内存问题
func (r *BalanceRepository) GetByChainPaginated(ctx context.Context, chainID string, offset, limit int) ([]models.UserBalance, error) {
	var balances []models.UserBalance
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Offset(offset).
		Limit(limit).
		Find(&balances).Error
	return balances, err
}

// CountByChain 返回指定链上有余额的用户总数
func (r *BalanceRepository) CountByChain(ctx context.Context, chainID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UserBalance{}).
		Where("chain_id = ?", chainID).
		Count(&count).Error
	return count, err
}

// GetBalancesAtTime 重建指定时间点的用户余额
// 使用余额历史确定给定时间戳的状态
func (r *BalanceRepository) GetBalancesAtTime(ctx context.Context, chainID string, timestamp int64) (map[string]string, error) {
	var histories []models.BalanceHistory
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT bh.* 
			FROM balance_history bh
			INNER JOIN (
				SELECT user_address, MAX(timestamp) as max_time
				FROM balance_history
				WHERE chain_id = ? AND timestamp <= FROM_UNIXTIME(?)
				GROUP BY user_address
			) latest ON bh.user_address = latest.user_address 
				AND bh.timestamp = latest.max_time
				AND bh.chain_id = ?
		`, chainID, timestamp, chainID).
		Scan(&histories).Error

	if err != nil {
		return nil, err
	}

	balances := make(map[string]string)
	for _, h := range histories {
		balances[h.UserAddress] = h.BalanceAfter
	}

	return balances, nil
}
