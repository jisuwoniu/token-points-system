package repository

import (
	"context"
	"errors"
	
	"gorm.io/gorm"
	"token-points-system/internal/models"
)

type BlockRepository struct {
	db *gorm.DB
}

func NewBlockRepository(db *gorm.DB) *BlockRepository {
	return &BlockRepository{db: db}
}

func (r *BlockRepository) GetLastProcessed(ctx context.Context, chainID string) (int64, error) {
	var block models.ProcessedBlock
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Order("block_number DESC").
		First(&block).Error
	
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return block.BlockNumber, err
}

func (r *BlockRepository) MarkProcessed(ctx context.Context, chainID string, blockNumber int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.ProcessedBlock
		err := tx.Where("chain_id = ?", chainID).First(&existing).Error
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			block := &models.ProcessedBlock{
				ChainID:     chainID,
				BlockNumber: blockNumber,
			}
			return tx.Create(block).Error
		}
		
		if err != nil {
			return err
		}
		
		return tx.Model(&existing).Update("block_number", blockNumber).Error
	})
}

func (r *BlockRepository) IsProcessed(ctx context.Context, chainID string, blockNumber int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ProcessedBlock{}).
		Where("chain_id = ? AND block_number = ?", chainID, blockNumber).
		Count(&count).Error
	return count > 0, err
}
