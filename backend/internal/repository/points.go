package repository

import (
	"context"
	"errors"
	
	"gorm.io/gorm"
	"token-points-system/internal/models"
)

type PointsRepository struct {
	db *gorm.DB
}

func NewPointsRepository(db *gorm.DB) *PointsRepository {
	return &PointsRepository{db: db}
}

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

func (r *PointsRepository) GetAllByChain(ctx context.Context, chainID string) ([]models.UserPoints, error) {
	var points []models.UserPoints
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Find(&points).Error
	return points, err
}

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
