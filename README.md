# Token Points System

基于ERC-20代币的积分系统，支持多链部署，实时追踪用户余额变动并计算积分。

## 📋 功能特性

- ✅ **ERC-20智能合约**：完整的mint和burn功能
- ✅ **多链支持**：支持Sepolia和Base Sepolia测试网
- ✅ **6区块确认机制**：防止区块链回滚导致的数据不一致
- ✅ **实时余额追踪**：精确记录每次余额变动
- ✅ **分钟级积分计算**：每小时自动计算用户积分
- ✅ **异常恢复机制**：支持回溯计算和数据备份
- ✅ **幂等性设计**：确保重复执行不会导致数据错误
- ✅ **清爽科技风格UI**：基于Bootstrap 5.3的现代化界面

## 🏗️ 系统架构

```
┌─────────────────┐
│  Smart Contract │ (ERC-20 with Mint/Burn)
└────────┬────────┘
         │ Events
         ▼
┌─────────────────┐
│  Go Backend     │
│  - Event Listener
│  - Balance Tracker
│  - Points Calculator
│  - Scheduler
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  MySQL 8.0.36   │
│  - Balances
│  - Points
│  - History
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  Bootstrap UI   │
└─────────────────┘
```

## 🚀 快速开始

### 前置要求

- Node.js 18+
- Go 1.21+
- MySQL 8.0.36
- MetaMask钱包（用于部署合约）

### 1. 数据库初始化

```bash
mysql -u root -p < database/schema.sql
```

### 2. 部署智能合约

```bash
cd contracts
npm install
cp .env.example .env
# 编辑.env文件，填入私钥和API密钥
npm run deploy:sepolia
npm run deploy:base-sepolia
```

部署完成后，将合约地址更新到 `backend/config/config.yaml` 中。

### 3. 启动后端服务（包含前端）

```bash
cd backend
go mod download
go run cmd/main.go
```

服务启动后，访问 http://localhost:8080 即可看到前端界面。

**注意**：前后端已合并为一个工程，后端服务会自动提供前端静态文件服务。

## 📊 数据库表结构

### 核心表

1. **user_balances** - 用户总余额表
2. **user_points** - 用户总积分表
3. **balance_history** - 余额变动记录表
4. **processed_blocks** - 区块处理记录表
5. **point_calculations** - 积分计算记录表（幂等性）
6. **calculation_backups** - 计算备份表

## 🔧 配置说明

### 后端配置 (backend/config/config.yaml)

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: your_password
  dbname: token_points_system

chains:
  - id: sepolia
    rpc_url: https://rpc.sepolia.org
    contract_address: "0x..."
    confirmation_blocks: 6
    
  - id: base-sepolia
    rpc_url: https://sepolia.base.org
    contract_address: "0x..."
    confirmation_blocks: 6

points:
  calculation_rate: 0.05
  calculation_interval: 3600
```

## 📈 积分计算规则

积分每小时计算一次，精确到分钟级别：

```
积分 = Σ (余额 × 0.05 × 持有分钟数 / 60)
```

**示例**：
- 15:00 余额为 0
- 15:10 增加到 100
- 15:30 增加到 200
- 16:00 计算积分

计算公式：
```
(100 × 0.05 × 20/60) + (200 × 0.05 × 30/60) = 1.67 + 5.00 = 6.67 积分
```

## 🔐 安全特性

- ✅ 6区块确认机制防止回滚
- ✅ 幂等性设计防止重复计算
- ✅ 事务处理确保数据一致性
- ✅ 错误处理和日志记录
- ✅ 定期数据备份

## 📝 API接口

### 查询余额
```
GET /api/balance/{chain}/{address}
```

### 查询积分
```
GET /api/points/{chain}/{address}
```

### 查询历史记录
```
GET /api/history/{chain}/{address}
```

### 触发回溯计算
```
POST /api/recalculate
{
  "chain": "sepolia",
  "startTime": "2024-01-01T00:00:00Z",
  "endTime": "2024-01-02T00:00:00Z"
}
```

### 创建备份
```
POST /api/backup
{
  "chain": "sepolia"
}
```

## 🛠️ 开发指南

### 核心技术实现

详细的技术实现说明请查看：[TECHNICAL_DETAILS.md](TECHNICAL_DETAILS.md)

**关键特性**：
- ✅ **定时拉取模式**：避免WebSocket订阅的消息丢失风险
- ✅ **协程池**：类似Java线程池的并发处理机制
- ✅ **自适应调节**：根据队列负载动态调整拉取频率
- ✅ **幂等性保证**：确保重复执行不会导致数据错误
- ✅ **监控指标**：实时监控系统运行状态

### 代码规范

- 单文件不超过160行
- 清晰的职责分离
- 统一的错误处理
- 完整的日志记录

### 项目结构

```
token-points-system/
├── contracts/          # 智能合约
├── backend/
│   ├── cmd/           # 入口文件
│   ├── internal/      # 内部包
│   │   ├── config/    # 配置
│   │   ├── models/    # 数据模型
│   │   ├── blockchain/# 区块链交互
│   │   ├── service/   # 业务逻辑
│   │   ├── repository/# 数据访问
│   │   └── scheduler/ # 定时任务
│   ├── pkg/           # 公共包
│   ├── web/           # 前端界面（已合并）
│   │   ├── index.html
│   │   ├── users.html
│   │   ├── admin.html
│   │   └── assets/
│   └── config/        # 配置文件
└── database/          # 数据库脚本
```

## 📄 License

MIT License

## 🤝 贡献

欢迎提交Issue和Pull Request！
