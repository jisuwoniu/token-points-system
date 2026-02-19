package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"token-points-system/internal/models"

	"gorm.io/gorm"
)

type CalculationRepository struct {
	db *gorm.DB
}

func NewCalculationRepository(db *gorm.DB) *CalculationRepository {
	return &CalculationRepository{db: db}
}

func (r *CalculationRepository) Create(ctx context.Context, calc *models.PointCalculation) error {
	return r.db.WithContext(ctx).Create(calc).Error
}

func (r *CalculationRepository) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.PointCalculation{}).
		Where("calculation_hash = ?", hash).
		Count(&count).Error
	return count > 0, err
}

func (r *CalculationRepository) GenerateHash(chainID, userAddress string, periodStart, periodEnd time.Time) string {
	data := fmt.Sprintf("%s:%s:%d:%d", chainID, userAddress, periodStart.Unix(), periodEnd.Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (r *CalculationRepository) GetByUser(ctx context.Context, chainID, userAddress string, limit int) ([]models.PointCalculation, error) {
	var calcs []models.PointCalculation
	query := r.db.WithContext(ctx).
		Where("chain_id = ? AND user_address = ?", chainID, userAddress).
		Order("period_end DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&calcs).Error
	return calcs, err
}

func (r *CalculationRepository) GetLastCalculation(ctx context.Context, chainID, userAddress string) (*models.PointCalculation, error) {
	var calc models.PointCalculation
	err := r.db.WithContext(ctx).
		Where("chain_id = ? AND user_address = ?", chainID, userAddress).
		Order("period_end DESC").
		First(&calc).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &calc, err
}

func (r *CalculationRepository) GetDailyPoints(ctx context.Context, days int) (map[string]float64, error) {
	type DailyPoints struct {
		Date  string
		Total float64
	}

	var results []DailyPoints

	err := r.db.WithContext(ctx).
		Model(&models.PointCalculation{}).
		Select("DATE_FORMAT(period_end, '%Y-%m-%d') as date, SUM(CAST(points_earned AS DECIMAL(65,18))) as total").
		Group("DATE_FORMAT(period_end, '%Y-%m-%d')").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	points := make(map[string]float64)
	for _, r := range results {
		points[r.Date] = r.Total
	}
	return points, nil
}
