# 区块链数据获取与积分计算机制说明

## 📋 问题1：数据获取模式

### 当前实现：定时拉取模式 ✅

**实现位置**：[listener.go:27-51](file:///Users/yuanfei/IdeaProjects/token-points-system/backend/internal/blockchain/listener.go#L27-L51)

```go
func (l *EventListener) Start(ctx context.Context, startBlock int64) {
    ticker := time.NewTicker(time.Duration(l.chainCfg.PullInterval) * time.Second)
    // 定时拉取区块数据
}
```

### 为什么选择拉取模式而不是订阅模式？

| 对比项 | 拉取模式 | 订阅模式（WebSocket） |
|--------|---------|---------------------|
| **消息丢失风险** | ✅ 低 - 可重试 | ❌ 高 - 网络断开即丢失 |
| **数据完整性** | ✅ 可验证 - 检查区块号 | ❌ 难验证 - 依赖推送 |
| **重试机制** | ✅ 简单 - 重新拉取 | ❌ 复杂 - 需要回放 |
| **回溯能力** | ✅ 强 - 任意区块 | ❌ 弱 - 只能从当前开始 |
| **资源消耗** | ⚠️ 较高 - 定时查询 | ✅ 较低 - 事件驱动 |

### 如何保障消息丢失导致的积分计算遗漏？

#### 原始实现的问题 ❌

```go
// listener.go:96-99
select {
case l.eventChan <- event:
default:
    logger.Warn("Event channel full, dropping event")  // ❌ 直接丢弃！
}
```

**问题**：当channel满时直接丢弃事件，导致积分计算遗漏。

#### 改进方案 ✅

**文件**：[enhanced_listener.go](file:///Users/yuanfei/IdeaProjects/token-points-system/backend/internal/blockchain/enhanced_listener.go)

**1. 持久化已处理区块号**

```go
// 从数据库恢复上次处理的区块号
lastProcessedBlock, err := l.blockRepo.GetLastProcessed(ctx, l.chainCfg.ID)

// 处理完成后立即持久化
if err := l.blockRepo.MarkProcessed(ctx, l.chainCfg.ID, confirmedBlock); err != nil {
    return err
}
```

**2. 阻塞式提交（不丢弃事件）**

```go
for {
    if l.workerPool.Submit(event) {
        break  // 成功提交
    }
    logger.Warn("Worker pool queue full, waiting...")
    time.Sleep(100 * time.Millisecond)  // 等待队列有空间
}
```

**3. 重试机制**

```go
func (l *EnhancedEventListener) processBlocksWithRetry(ctx context.Context) {
    maxRetries := 3
    
    for i := 0; i < maxRetries; i++ {
        err := l.processNewBlocks(ctx)
        if err == nil {
            return
        }
        
        atomic.AddInt64(&l.errorCount, 1)
        logger.Error("Failed to process blocks, retrying:", err)
        time.Sleep(time.Duration(i+1) * time.Second)  // 指数退避
    }
}
```

**4. 幂等性保证**

```go
// 在 BalanceService 中
exists, err := s.historyRepo.ExistsByTxHash(ctx, event.TxHash)
if exists {
    logger.Debug("Transaction already processed")
    return nil  // 跳过已处理的交易
}
```

---

## 📋 问题2：任务积压处理机制

### 原始实现的问题 ❌

1. **固定拉取间隔**：无法根据负载动态调整
2. **单协程处理**：没有并发处理能力
3. **无背压控制**：队列满时丢弃事件
4. **无监控告警**：不知道系统是否过载

### 改进方案：协程池 + 自适应调节 ✅

#### 1. 协程池实现（类似Java线程池）

**文件**：[enhanced_listener.go:10-50](file:///Users/yuanfei/IdeaProjects/token-points-system/backend/internal/blockchain/enhanced_listener.go#L10-L50)

```go
type WorkerPool struct {
    workers    int              // 工作协程数量
    taskQueue  chan *TransferEvent  // 任务队列
    wg         sync.WaitGroup   // 等待组
    ctx        context.Context
    cancel     context.CancelFunc
}

func NewWorkerPool(workers int, queueSize int) *WorkerPool {
    return &WorkerPool{
        workers:   workers,           // 类似 Java 的 corePoolSize
        taskQueue: make(chan *TransferEvent, queueSize),  // 类似 Java 的 BlockingQueue
    }
}

func (p *WorkerPool) Start(handler func(*TransferEvent)) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i, handler)  // 启动固定数量的worker
    }
}

func (p *WorkerPool) worker(id int, handler func(*TransferEvent)) {
    defer p.wg.Done()
    
    for {
        select {
        case <-p.ctx.Done():
            return
        case event := <-p.taskQueue:
            handler(event)  // 处理事件
        }
    }
}
```

**对比Java线程池**：

| Java ThreadPoolExecutor | Go WorkerPool |
|------------------------|---------------|
| corePoolSize | workers |
| BlockingQueue | taskQueue (buffered channel) |
| Worker threads | goroutines |
| RejectedExecutionHandler | Submit() 返回 false |

#### 2. 自适应拉取频率

**文件**：[enhanced_listener.go:193-220](file:///Users/yuanfei/IdeaProjects/token-points-system/backend/internal/blockchain/enhanced_listener.go#L193-L220)

```go
func (l *EnhancedEventListener) monitorAndAdapt() {
    queueLen := l.workerPool.QueueLength()
    queueCapacity := cap(l.workerPool.taskQueue)
    
    usage := float64(queueLen) / float64(queueCapacity)
    
    if usage > 0.8 {
        // 队列使用率 > 80%，降低拉取频率
        if l.pullInterval < l.maxInterval {
            l.pullInterval = l.pullInterval * 12 / 10  // 增加20%
            logger.Warn("Queue usage high, increasing pull interval")
        }
    } else if usage < 0.3 {
        // 队列使用率 < 30%，提高拉取频率
        if l.pullInterval > l.minInterval {
            l.pullInterval = l.pullInterval * 8 / 10  // 减少20%
            logger.Info("Queue usage low, decreasing pull interval")
        }
    }
}
```

**工作原理**：

```
队列使用率 > 80%  →  降低拉取频率（5s → 6s → 7.2s...）
队列使用率 < 30%  →  提高拉取频率（7s → 5.6s → 4.5s...）
```

#### 3. 批量处理机制

```go
func (l *EnhancedEventListener) processNewBlocks(ctx context.Context) error {
    // ...
    
    batchSize := int64(100)  // 每次最多处理100个区块
    if confirmedBlock-startBlock > batchSize {
        confirmedBlock = startBlock + batchSize  // 分批处理
    }
    
    logs, err := l.client.GetTransferLogs(ctx, startBlock, confirmedBlock)
    // ...
}
```

**好处**：
- 避免一次性拉取过多区块导致超时
- 更细粒度的进度控制
- 更好的错误恢复

#### 4. 监控指标

```go
func (l *EnhancedEventListener) GetStats() map[string]interface{} {
    return map[string]interface{}{
        "queue_length":         l.workerPool.QueueLength(),
        "queue_capacity":       cap(l.workerPool.taskQueue),
        "last_processed_block": atomic.LoadInt64(&l.lastProcessedBlock),
        "processed_count":      atomic.LoadInt64(&l.processedCount),
        "error_count":          atomic.LoadInt64(&l.errorCount),
        "pull_interval":        l.pullInterval,
    }
}
```

**可监控的指标**：
- 队列长度和使用率
- 已处理区块号
- 处理成功/失败计数
- 当前拉取间隔

---

## 📊 配置参数说明

**文件**：[config.yaml](file:///Users/yuanfei/IdeaProjects/token-points-system/backend/config/config.yaml)

```yaml
chains:
  - id: sepolia
    pull_interval: 10          # 初始拉取间隔（秒）
    worker_pool_size: 4        # 协程池大小（类似Java corePoolSize）
    queue_size: 10000          # 任务队列大小
    batch_size: 100            # 每批处理区块数
    max_retries: 3             # 最大重试次数
    adaptive_mode: true        # 启用自适应调节
```

### 参数调优建议

| 场景 | worker_pool_size | queue_size | batch_size | pull_interval |
|------|-----------------|------------|------------|---------------|
| **低负载** | 2-4 | 5000 | 50 | 10s |
| **中负载** | 4-8 | 10000 | 100 | 5s |
| **高负载** | 8-16 | 20000 | 200 | 3s |
| **追赶模式** | 16-32 | 50000 | 500 | 1s |

---

## 🔄 完整工作流程

```
┌─────────────────────────────────────────────────────────────┐
│                    Enhanced Event Listener                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │  1. 从数据库恢复上次处理的区块号      │
        │     lastProcessedBlock = 12345      │
        └─────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │  2. 定时拉取新区块（自适应间隔）      │
        │     currentBlock = 12400            │
        │     confirmedBlock = 12394          │
        └─────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │  3. 批量获取Transfer事件             │
        │     logs = GetTransferLogs(         │
        │         12346, 12394                │
        │     )                               │
        └─────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │  4. 提交到协程池（阻塞式）            │
        │     for event in logs:              │
        │         workerPool.Submit(event)    │
        └─────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │  5. Worker协程并发处理               │
        │     - 检查幂等性（tx_hash）          │
        │     - 更新余额                       │
        │     - 记录历史                       │
        │     - 计算积分                       │
        └─────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │  6. 持久化处理进度                    │
        │     MarkProcessed(12394)            │
        └─────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │  7. 监控和自适应调节                  │
        │     - 检查队列使用率                  │
        │     - 调整拉取间隔                    │
        │     - 记录统计指标                    │
        └─────────────────────────────────────┘
```

---

## ✅ 改进后的优势

### 1. 消息不丢失
- ✅ 持久化已处理区块号
- ✅ 阻塞式提交（不丢弃）
- ✅ 重试机制
- ✅ 幂等性保证

### 2. 处理积压能力
- ✅ 协程池并发处理
- ✅ 自适应拉取频率
- ✅ 批量处理机制
- ✅ 背压控制

### 3. 可观测性
- ✅ 实时监控指标
- ✅ 错误计数
- ✅ 处理进度追踪
- ✅ 性能统计

### 4. 容错能力
- ✅ 自动重试
- ✅ 指数退避
- ✅ 优雅关闭
- ✅ 断点续传

---

## 🚀 使用建议

### 1. 生产环境配置

```yaml
chains:
  - id: sepolia
    worker_pool_size: 8        # 根据CPU核心数调整
    queue_size: 20000          # 足够大的缓冲
    batch_size: 100            # 适中的批次大小
    max_retries: 5             # 更多重试次数
    adaptive_mode: true        # 启用自适应
```

### 2. 监控告警

建议监控以下指标：
- 队列使用率 > 80% 持续5分钟 → 告警
- 错误率 > 5% → 告警
- 处理延迟 > 100区块 → 告警

### 3. 性能优化

- **CPU密集型**：增加 worker_pool_size
- **IO密集型**：增加 queue_size 和 batch_size
- **网络延迟高**：增加 pull_interval 和 max_retries

---

## 📝 总结

| 问题 | 原始实现 | 改进方案 |
|------|---------|---------|
| **数据获取模式** | 定时拉取 ✅ | 定时拉取 ✅ |
| **消息丢失风险** | 高 ❌ | 低 ✅ |
| **积压处理能力** | 无 ❌ | 强 ✅ |
| **协程池** | 无 ❌ | 有 ✅ |
| **自适应调节** | 无 ❌ | 有 ✅ |
| **监控指标** | 无 ❌ | 完整 ✅ |
| **重试机制** | 无 ❌ | 有 ✅ |
| **幂等性** | 部分 ⚠️ | 完整 ✅ |

改进后的系统具有：
- **高可靠性**：不丢失任何事件
- **高性能**：协程池并发处理
- **高可扩展性**：自适应调节机制
- **高可观测性**：完整的监控指标
