package models

import (
	"time"
)

type ChangeType string

const (
	ChangeTypeTransfer ChangeType = "transfer"
	ChangeTypeMint     ChangeType = "mint"
	ChangeTypeBurn     ChangeType = "burn"
)

type BalanceHistory struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID       string    `gorm:"size:50;not null;index:idx_chain_user_time" json:"chain_id"`
	UserAddress   string    `gorm:"size:42;not null;index:idx_chain_user_time" json:"user_address"`
	BalanceBefore string    `gorm:"type:decimal(65,0);not null" json:"balance_before"`
	BalanceAfter  string    `gorm:"type:decimal(65,0);not null" json:"balance_after"`
	ChangeAmount  string    `gorm:"type:decimal(65,0);not null" json:"change_amount"`
	ChangeType    ChangeType `gorm:"type:enum('transfer','mint','burn');not null" json:"change_type"`
	TxHash        string    `gorm:"size:66;not null;uniqueIndex:uk_tx" json:"tx_hash"`
	BlockNumber   int64     `gorm:"not null;index" json:"block_number"`
	Timestamp     time.Time `gorm:"not null;index:idx_chain_user_time" json:"timestamp"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (BalanceHistory) TableName() string {
	return "balance_history"
}
