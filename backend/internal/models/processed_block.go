package models

import (
	"time"
)

type ProcessedBlock struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID     string    `gorm:"uniqueIndex:uk_chain_block;size:50;not null" json:"chain_id"`
	BlockNumber int64     `gorm:"uniqueIndex:uk_chain_block;not null" json:"block_number"`
	ProcessedAt time.Time `gorm:"autoCreateTime" json:"processed_at"`
}

func (ProcessedBlock) TableName() string {
	return "processed_blocks"
}
