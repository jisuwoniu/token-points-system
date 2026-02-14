package repository

import (
	"context"
	"errors"

	"token-points-system/internal/models"

	"gorm.io/gorm"
)

type PointsRepository struct {
	db *gorm.DB
}

func NewPointsRepository(db *gorm.DB) *PointsRepository {
	return &PointsRepository{db: db}
}

// GetByUser 获取指定用户在指定链上的总积分
func (r *PointsRepository) GetByUser(ctx context.Context, chainID, userAddress string) (*models.UserPoints, error) {
	var points models.UserPoints
	err := r.db.WithContext(ctx).
		Where("chain_id = ? AND user_address = ?", chainID, userAddress).
		First(&points).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &points, err
}

// UpdatePoints 更新或创建用户积分记录
func (r *PointsRepository) UpdatePoints(ctx context.Context, chainID, userAddress, totalPoints string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		points := &models.UserPoints{
			ChainID:     chainID,
			UserAddress: userAddress,
			TotalPoints: totalPoints,
		}

		result := tx.Where("chain_id = ? AND user_address = ?", chainID, userAddress).
			Assign(points).
			FirstOrCreate(points)

		return result.Error
	})
}

// GetAllByChain 获取指定链上所有用户的积分
// 警告：可能返回大量数据，生产环境请使用GetByChainPaginated
func (r *PointsRepository) GetAllByChain(ctx context.Context, chainID string) ([]models.UserPoints, error) {
	var points []models.UserPoints
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Find(&points).Error
	return points, err
}

// GetByChainPaginated 分页获取用户积分
// 生产环境使用此方法避免大数据集导致的内存问题
func (r *PointsRepository) GetByChainPaginated(ctx context.Context, chainID string, offset, limit int) ([]models.UserPoints, error) {
	var points []models.UserPoints
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Offset(offset).
		Limit(limit).
		Find(&points).Error
	return points, err
}

// CountByChain 返回指定链上有积分的用户总数
func (r *PointsRepository) CountByChain(ctx context.Context, chainID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UserPoints{}).
		Where("chain_id = ?", chainID).
		Count(&count).Error
	return count, err
}

// AddPoints 原子性增加用户积分
// 使用INSERT ... ON DUPLICATE KEY UPDATE实现upsert
func (r *PointsRepository) AddPoints(ctx context.Context, chainID, userAddress, pointsEarned string) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO user_points (chain_id, user_address, total_points, last_calculated_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE 
			total_points = total_points + ?,
			last_calculated_at = NOW(),
			updated_at = NOW()
	`, chainID, userAddress, pointsEarned, pointsEarned).Error
}
