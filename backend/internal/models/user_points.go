package models

import (
	"time"

	"gorm.io/gorm"
)

type UserPoints struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID           string         `gorm:"uniqueIndex:uk_chain_user;size:50;not null" json:"chain_id"`
	UserAddress       string         `gorm:"uniqueIndex:uk_chain_user;size:42;not null" json:"user_address"`
	TotalPoints       string         `gorm:"type:decimal(65,18);not null;default:0" json:"total_points"`
	LastCalculatedAt  *time.Time     `json:"last_calculated_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (UserPoints) TableName() string {
	return "user_points"
}
