# 区块处理流程改进方案

## 问题分析

### 当前问题

1. **事件处理和区块标记不同步**
   - listener 返回 confirmedBlock 后立即更新 lastProcessedBlock
   - 但事件还在 channel 中等待处理
   - 中断后会导致事件丢失

2. **MarkProcessed 调用时机错误**
   - 每处理一个事件就调用 MarkProcessed
   - 一个区块可能有多个事件
   - 第一个事件处理完就标记，后续事件失败会丢失

## 改进方案

### 方案1：批量处理 + 延迟标记（推荐）

#### 流程

```
1. 获取区块 [100-200] 的所有事件
2. 处理所有事件（事务）
3. 所有事件处理成功后，标记区块 200 已处理
4. 更新 lastProcessedBlock = 200
```

#### 实现

```go
// 改进后的 listener
func (l *EventListener) processNewBlocks(ctx context.Context, lastBlock int64) (int64, error) {
    confirmedBlock, err := l.client.GetConfirmBlockNumber(ctx)
    if err != nil {
        return lastBlock, err
    }
    
    if confirmedBlock <= lastBlock {
        return lastBlock, nil
    }
    
    startBlock := lastBlock + 1
    
    // 限制批次大小
    batchSize := int64(100)
    if confirmedBlock-startBlock > batchSize {
        confirmedBlock = startBlock + batchSize
    }
    
    logs, err := l.client.GetTransferLogs(ctx, startBlock, confirmedBlock)
    if err != nil {
        return lastBlock, err
    }
    
    // 批量处理所有事件
    for _, log := range logs {
        event, err := ParseTransferLog(log)
        if err != nil {
            logger.Error("Failed to parse log:", err)
            continue
        }
        
        // 同步处理事件（不使用 channel）
        if err := l.balanceSvc.ProcessTransfer(ctx, l.chainCfg.ID, event, timestamp); err != nil {
            logger.Error("Failed to process transfer:", err)
            return lastBlock, err  // 失败时不更新区块号
        }
    }
    
    // 所有事件处理成功后，才返回新的区块号
    return confirmedBlock, nil
}
```

### 方案2：事件级别幂等性 + 区块级别标记

#### 流程

```
1. 获取区块 [100-200] 的所有事件
2. 逐个处理事件（通过 tx_hash 去重）
3. 所有事件处理完成后，标记区块 200 已处理
4. 更新 lastProcessedBlock = 200
```

#### 关键改进

1. **删除 ProcessTransfer 中的 MarkProcessed**
   ```go
   func (s *BalanceService) ProcessTransfer(...) error {
       // 处理事件
       // 删除：return s.blockRepo.MarkProcessed(ctx, chainID, event.BlockNum)
       return nil
   }
   ```

2. **在 listener 中统一标记**
   ```go
   func (l *EventListener) Start(ctx context.Context, startBlock int64) {
       lastProcessedBlock := startBlock
       
       for {
           select {
           case <-ticker.C:
               newBlock, err := l.processNewBlocks(ctx, lastProcessedBlock)
               if err != nil {
                   logger.Error("Failed to process blocks:", err)
                   continue
               }
               
               // 只有成功处理所有事件后才更新
               if newBlock > lastProcessedBlock {
                   if err := l.blockRepo.MarkProcessed(ctx, l.chainCfg.ID, newBlock); err != nil {
                       logger.Error("Failed to mark processed:", err)
                       continue
                   }
                   lastProcessedBlock = newBlock
               }
           }
       }
   }
   ```

### 方案3：使用数据库事务保证原子性

#### 实现

```go
func (s *BalanceService) ProcessBlockEvents(ctx context.Context, chainID string, events []*blockchain.TransferEvent) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        for _, event := range events {
            // 检查是否已处理
            exists, err := s.historyRepo.ExistsByTxHashWithTx(tx, event.TxHash)
            if err != nil {
                return err
            }
            if exists {
                continue
            }
            
            // 处理事件
            if err := s.processUserTransferWithTx(tx, chainID, event); err != nil {
                return err
            }
        }
        
        // 标记区块已处理
        if len(events) > 0 {
            lastBlock := events[len(events)-1].BlockNum
            if err := s.blockRepo.MarkProcessedWithTx(tx, chainID, lastBlock); err != nil {
                return err
            }
        }
        
        return nil
    })
}
```

## 推荐方案

**推荐使用方案1 + 方案3的组合**：

1. 批量获取区块事件
2. 使用数据库事务处理所有事件
3. 事务成功后标记区块已处理
4. 失败时不更新区块号，下次重新处理

## 关键改进点

1. ✅ 删除 ProcessTransfer 中的 MarkProcessed
2. ✅ 在 listener 中统一标记区块
3. ✅ 使用事务保证原子性
4. ✅ 通过 tx_hash 保证幂等性
5. ✅ 失败时不更新区块号

## 数据一致性保证

- **幂等性**：通过 tx_hash 去重
- **原子性**：使用数据库事务
- **可恢复性**：失败时不更新区块号
- **完整性**：批量处理，确保不遗漏
