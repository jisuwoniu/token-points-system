package repository

import (
	"context"
	"errors"
	
	"gorm.io/gorm"
	"token-points-system/internal/models"
)

type BalanceRepository struct {
	db *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

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

func (r *BalanceRepository) GetAllByChain(ctx context.Context, chainID string) ([]models.UserBalance, error) {
	var balances []models.UserBalance
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Find(&balances).Error
	return balances, err
}

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
