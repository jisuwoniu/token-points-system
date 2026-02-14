package blockchain

import (
	"context"
	"sync/atomic"
	"time"

	"token-points-system/internal/config"
	"token-points-system/internal/repository"
	"token-points-system/pkg/logger"
)

type EventListener struct {
	chainCfg     *config.ChainConfig
	client       *Client
	blockRepo    *repository.BlockRepository
	eventChan    chan *TransferEvent
	stopChan     chan struct{}
	isProcessing int32
}

func NewEventListener(chainCfg *config.ChainConfig, client *Client, blockRepo *repository.BlockRepository) *EventListener {
	return &EventListener{
		chainCfg:  chainCfg,
		client:    client,
		blockRepo: blockRepo,
		eventChan: make(chan *TransferEvent, 1000),
		stopChan:  make(chan struct{}),
	}
}

// Start 启动事件监听器
func (l *EventListener) Start(ctx context.Context, startBlock int64) {
	ticker := time.NewTicker(time.Duration(l.chainCfg.PullInterval) * time.Second)
	defer ticker.Stop()

	lastProcessedBlock := startBlock

	for {
		select {
		case <-ctx.Done():
			logger.Info("事件监听器已停止：上下文已取消")
			return
		case <-l.stopChan:
			logger.Info("事件监听器已停止：收到停止信号")
			return
		case <-ticker.C:
			// 检查是否正在处理
			if atomic.LoadInt32(&l.isProcessing) == 1 {
				logger.WithFields(map[string]interface{}{
					"chain_id": l.chainCfg.ID,
				}).Warn("上一次处理尚未完成，跳过本次触发")
				continue
			}

			// 标记为处理中
			atomic.StoreInt32(&l.isProcessing, 1)

			// 处理新区块
			block, err := l.processNewBlocks(ctx, lastProcessedBlock)
			if err != nil {
				logger.Error("处理区块失败:", err)
			} else if block > lastProcessedBlock {
				lastProcessedBlock = block
			}

			// 标记为空闲
			atomic.StoreInt32(&l.isProcessing, 0)
		}
	}
}

// Stop 停止事件监听器
func (l *EventListener) Stop() {
	close(l.stopChan)
}

// GetEventChannel 获取事件通道
func (l *EventListener) GetEventChannel() <-chan *TransferEvent {
	return l.eventChan
}

// IsProcessing 返回是否正在处理
func (l *EventListener) IsProcessing() bool {
	return atomic.LoadInt32(&l.isProcessing) == 1
}

// processNewBlocks 处理新区块
func (l *EventListener) processNewBlocks(ctx context.Context, lastBlock int64) (int64, error) {
	confirmedBlock, err := l.client.GetConfirmBlockNumber(ctx)
	if err != nil {
		return lastBlock, err
	}

	if confirmedBlock <= lastBlock {
		return lastBlock, nil
	}

	startBlock := lastBlock + 1
	if startBlock == 1 && l.chainCfg.StartBlock > 0 {
		startBlock = l.chainCfg.StartBlock
	}

	batchSize := int64(l.chainCfg.BatchSize)
	if batchSize <= 0 {
		batchSize = 100
	}

	maxBatchSize := int64(5000)
	if batchSize > maxBatchSize {
		batchSize = maxBatchSize
	}

	if confirmedBlock-startBlock >= batchSize {
		confirmedBlock = startBlock + batchSize - 1
	}

	logger.WithFields(map[string]interface{}{
		"chain_id":        l.chainCfg.ID,
		"start_block":     startBlock,
		"confirmed_block": confirmedBlock,
		"batch_size":      batchSize,
		"is_processing":   l.IsProcessing(),
	}).Info("处理新区块")

	logs, err := l.client.GetTransferLogs(ctx, startBlock, confirmedBlock)
	if err != nil {
		return lastBlock, err
	}

	// 即使没有事件，也要标记区块已处理
	if len(logs) == 0 {
		logger.WithFields(map[string]interface{}{
			"chain_id":        l.chainCfg.ID,
			"start_block":     startBlock,
			"confirmed_block": confirmedBlock,
		}).Debug("区块范围内无Transfer事件")

		// 标记区块已处理，避免重复拉取
		if err := l.blockRepo.MarkProcessed(ctx, l.chainCfg.ID, confirmedBlock); err != nil {
			logger.Error("标记区块已处理失败:", err)
			return lastBlock, err
		}

		return confirmedBlock, nil
	}

	// 有事件时，发送到通道
	for _, log := range logs {
		event, err := ParseTransferLog(log)
		if err != nil {
			logger.Error("解析日志失败:", err)
			continue
		}

		select {
		case l.eventChan <- event:
		default:
			logger.Warn("事件通道已满，丢弃事件")
		}
	}

	// 注意：有事件时不在这里标记，由ProcessTransfer处理完成后标记
	// 这样可以确保事件处理完成后才更新区块号

	return confirmedBlock, nil
}
