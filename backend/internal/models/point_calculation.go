package models

import (
	"time"
)

type PointCalculation struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID         string    `gorm:"size:50;not null;index:idx_chain_user_period" json:"chain_id"`
	UserAddress     string    `gorm:"size:42;not null;index:idx_chain_user_period" json:"user_address"`
	PeriodStart     time.Time `gorm:"not null;index:idx_chain_user_period" json:"period_start"`
	PeriodEnd       time.Time `gorm:"not null;index:idx_chain_user_period" json:"period_end"`
	PointsEarned    string    `gorm:"type:decimal(65,18);not null" json:"points_earned"`
	CalculationHash string    `gorm:"size:64;not null;uniqueIndex" json:"calculation_hash"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PointCalculation) TableName() string {
	return "point_calculations"
}
