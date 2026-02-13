package blockchain

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"token-points-system/internal/config"
	"token-points-system/internal/repository"
	"token-points-system/pkg/logger"
)

type WorkerPool struct {
	workers    int
	taskQueue  chan *TransferEvent
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:   workers,
		taskQueue: make(chan *TransferEvent, queueSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (p *WorkerPool) Start(handler func(*TransferEvent)) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i, handler)
	}
}

func (p *WorkerPool) worker(id int, handler func(*TransferEvent)) {
	defer p.wg.Done()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case event := <-p.taskQueue:
			handler(event)
		}
	}
}

func (p *WorkerPool) Submit(event *TransferEvent) bool {
	select {
	case p.taskQueue <- event:
		return true
	default:
		return false
	}
}

func (p *WorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
}

func (p *WorkerPool) QueueLength() int {
	return len(p.taskQueue)
}

type EnhancedEventListener struct {
	chainCfg    *config.ChainConfig
	client      *Client
	blockRepo   *repository.BlockRepository
	workerPool  *WorkerPool
	stopChan    chan struct{}
	
	mu                sync.RWMutex
	lastProcessedBlock int64
	pullInterval      time.Duration
	
	adaptiveMode      bool
	minInterval       time.Duration
	maxInterval       time.Duration
	
	processedCount    int64
	errorCount        int64
}

func NewEnhancedEventListener(
	chainCfg *config.ChainConfig,
	client *Client,
	blockRepo *repository.BlockRepository,
) *EnhancedEventListener {
	workers := 4
	queueSize := 10000
	
	return &EnhancedEventListener{
		chainCfg:    chainCfg,
		client:      client,
		blockRepo:   blockRepo,
		workerPool:  NewWorkerPool(workers, queueSize),
		stopChan:    make(chan struct{}),
		pullInterval: time.Duration(chainCfg.PullInterval) * time.Second,
		adaptiveMode: true,
		minInterval:  5 * time.Second,
		maxInterval:  60 * time.Second,
	}
}

func (l *EnhancedEventListener) Start(ctx context.Context, startBlock int64) {
	lastProcessedBlock, err := l.blockRepo.GetLastProcessed(ctx, l.chainCfg.ID)
	if err != nil {
		logger.Error("Failed to get last processed block:", err)
		lastProcessedBlock = startBlock
	}
	
	l.lastProcessedBlock = lastProcessedBlock
	
	l.workerPool.Start(func(event *TransferEvent) {
		atomic.AddInt64(&l.processedCount, 1)
	})
	
	ticker := time.NewTicker(l.pullInterval)
	defer ticker.Stop()
	
	monitorTicker := time.NewTicker(10 * time.Second)
	defer monitorTicker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			l.workerPool.Stop()
			return
		case <-l.stopChan:
			l.workerPool.Stop()
			return
		case <-ticker.C:
			l.processBlocksWithRetry(ctx)
		case <-monitorTicker.C:
			l.monitorAndAdapt()
		}
	}
}

func (l *EnhancedEventListener) processBlocksWithRetry(ctx context.Context) {
	maxRetries := 3
	
	for i := 0; i < maxRetries; i++ {
		err := l.processNewBlocks(ctx)
		if err == nil {
			return
		}
		
		atomic.AddInt64(&l.errorCount, 1)
		logger.Error("Failed to process blocks, retrying:", err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}
}

func (l *EnhancedEventListener) processNewBlocks(ctx context.Context) error {
	confirmedBlock, err := l.client.GetConfirmBlockNumber(ctx)
	if err != nil {
		return err
	}
	
	l.mu.RLock()
	lastBlock := l.lastProcessedBlock
	l.mu.RUnlock()
	
	if confirmedBlock <= lastBlock {
		return nil
	}
	
	startBlock := lastBlock + 1
	if startBlock == 1 && l.chainCfg.StartBlock > 0 {
		startBlock = l.chainCfg.StartBlock
	}
	
	batchSize := int64(100)
	if confirmedBlock-startBlock > batchSize {
		confirmedBlock = startBlock + batchSize
	}
	
	logs, err := l.client.GetTransferLogs(ctx, startBlock, confirmedBlock)
	if err != nil {
		return err
	}
	
	for _, log := range logs {
		event, err := ParseTransferLog(log)
		if err != nil {
			logger.Error("Failed to parse log:", err)
			continue
		}
		
		for {
			if l.workerPool.Submit(event) {
				break
			}
			logger.Warn("Worker pool queue full, waiting...")
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	if err := l.blockRepo.MarkProcessed(ctx, l.chainCfg.ID, confirmedBlock); err != nil {
		return err
	}
	
	l.mu.Lock()
	l.lastProcessedBlock = confirmedBlock
	l.mu.Unlock()
	
	logger.WithFields(map[string]interface{}{
		"chain_id":        l.chainCfg.ID,
		"start_block":     startBlock,
		"confirmed_block": confirmedBlock,
		"logs_count":      len(logs),
	}).Info("Processed blocks")
	
	return nil
}

func (l *EnhancedEventListener) monitorAndAdapt() {
	queueLen := l.workerPool.QueueLength()
	queueCapacity := cap(l.workerPool.taskQueue)
	
	usage := float64(queueLen) / float64(queueCapacity)
	
	if !l.adaptiveMode {
		return
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if usage > 0.8 {
		if l.pullInterval < l.maxInterval {
			l.pullInterval = l.pullInterval * 12 / 10
			logger.WithFields(map[string]interface{}{
				"queue_usage":    usage,
				"new_interval":   l.pullInterval,
			}).Warn("Queue usage high, increasing pull interval")
		}
	} else if usage < 0.3 {
		if l.pullInterval > l.minInterval {
			l.pullInterval = l.pullInterval * 8 / 10
			logger.WithFields(map[string]interface{}{
				"queue_usage":    usage,
				"new_interval":   l.pullInterval,
			}).Info("Queue usage low, decreasing pull interval")
		}
	}
}

func (l *EnhancedEventListener) Stop() {
	close(l.stopChan)
}

func (l *EnhancedEventListener) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"queue_length":        l.workerPool.QueueLength(),
		"queue_capacity":      cap(l.workerPool.taskQueue),
		"last_processed_block": atomic.LoadInt64(&l.lastProcessedBlock),
		"processed_count":     atomic.LoadInt64(&l.processedCount),
		"error_count":         atomic.LoadInt64(&l.errorCount),
		"pull_interval":       l.pullInterval,
	}
}
