package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type BackupType string

const (
	BackupTypeBalanceSnapshot BackupType = "balance_snapshot"
	BackupTypePointsSnapshot  BackupType = "points_snapshot"
	BackupTypeCalcState       BackupType = "calculation_state"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

type CalculationBackup struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID     string    `gorm:"size:50;not null;index:idx_chain_type_time" json:"chain_id"`
	BackupType  BackupType `gorm:"type:enum('balance_snapshot','points_snapshot','calculation_state');not null;index:idx_chain_type_time" json:"backup_type"`
	BackupData  JSONB     `gorm:"type:json;not null" json:"backup_data"`
	CreatedAt   time.Time `gorm:"autoCreateTime;index:idx_chain_type_time" json:"created_at"`
}

func (CalculationBackup) TableName() string {
	return "calculation_backups"
}
