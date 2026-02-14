package repository

import (
	"context"
	"errors"

	"token-points-system/internal/models"

	"gorm.io/gorm"
)

type BlockRepository struct {
	db *gorm.DB
}

func NewBlockRepository(db *gorm.DB) *BlockRepository {
	return &BlockRepository{db: db}
}

// GetLastProcessed 获取指定链最后处理的区块号
// 如果没有处理过区块，返回0
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

// UpdateLastProcessed 更新或创建指定链的处理区块记录
// 使用upsert保持每条链只有一条记录
func (r *BlockRepository) UpdateLastProcessed(ctx context.Context, chainID string, blockNumber int64) error {
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

// MarkProcessed 标记区块已处理
// 已废弃：请使用UpdateLastProcessed以获得更好的性能
func (r *BlockRepository) MarkProcessed(ctx context.Context, chainID string, blockNumber int64) error {
	return r.UpdateLastProcessed(ctx, chainID, blockNumber)
}

// IsProcessed 检查区块是否已处理
// 注意：检查区块号是否<=最后处理的区块号
func (r *BlockRepository) IsProcessed(ctx context.Context, chainID string, blockNumber int64) (bool, error) {
	var block models.ProcessedBlock
	err := r.db.WithContext(ctx).
		Where("chain_id = ? AND block_number >= ?", chainID, blockNumber).
		First(&block).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

// GetProcessingStats 获取指定链的处理统计信息
func (r *BlockRepository) GetProcessingStats(ctx context.Context, chainID string) (map[string]interface{}, error) {
	var block models.ProcessedBlock
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		First(&block).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]interface{}{
			"chain_id":     chainID,
			"last_block":   0,
			"processed_at": nil,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"chain_id":     block.ChainID,
		"last_block":   block.BlockNumber,
		"processed_at": block.ProcessedAt,
	}, nil
}
