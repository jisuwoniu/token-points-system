package blockchain

import (
	"context"
	"time"

	"token-points-system/internal/config"
	"token-points-system/pkg/logger"
)

type EventListener struct {
	chainCfg  *config.ChainConfig
	client    *Client
	eventChan chan *TransferEvent
	stopChan  chan struct{}
}

func NewEventListener(chainCfg *config.ChainConfig, client *Client) *EventListener {
	return &EventListener{
		chainCfg:  chainCfg,
		client:    client,
		eventChan: make(chan *TransferEvent, 1000),
		stopChan:  make(chan struct{}),
	}
}

func (l *EventListener) Start(ctx context.Context, startBlock int64) {
	ticker := time.NewTicker(time.Duration(l.chainCfg.PullInterval) * time.Second)
	defer ticker.Stop()
	
	lastProcessedBlock := startBlock
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Event listener stopped: context cancelled")
			return
		case <-l.stopChan:
			logger.Info("Event listener stopped: stop signal received")
			return
		case <-ticker.C:
			block, err := l.processNewBlocks(ctx, lastProcessedBlock)
			if err != nil {
				logger.Error("Failed to process blocks:", err)
				continue
			}
			if block > lastProcessedBlock {
				lastProcessedBlock = block
			}
		}
	}
}

func (l *EventListener) Stop() {
	close(l.stopChan)
}

func (l *EventListener) GetEventChannel() <-chan *TransferEvent {
	return l.eventChan
}

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
	
	logger.WithFields(map[string]interface{}{
		"chain_id":        l.chainCfg.ID,
		"start_block":     startBlock,
		"confirmed_block": confirmedBlock,
	}).Info("Processing new blocks")
	
	logs, err := l.client.GetTransferLogs(ctx, startBlock, confirmedBlock)
	if err != nil {
		return lastBlock, err
	}
	
	for _, log := range logs {
		event, err := ParseTransferLog(log)
		if err != nil {
			logger.Error("Failed to parse log:", err)
			continue
		}
		
		select {
		case l.eventChan <- event:
		default:
			logger.Warn("Event channel full, dropping event")
		}
	}
	
	return confirmedBlock, nil
}
