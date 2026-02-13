package models

import (
	"time"

	"gorm.io/gorm"
)

type UserBalance struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID     string         `gorm:"uniqueIndex:uk_chain_user;size:50;not null" json:"chain_id"`
	UserAddress string         `gorm:"uniqueIndex:uk_chain_user;size:42;not null" json:"user_address"`
	Balance     string         `gorm:"type:decimal(65,0);not null;default:0" json:"balance"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (UserBalance) TableName() string {
	return "user_balances"
}
