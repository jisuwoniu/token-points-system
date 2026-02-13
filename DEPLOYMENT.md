# 部署指南

## 环境准备

### 1. 安装依赖软件

```bash
# Node.js 18+
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Go 1.21+
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# MySQL 8.0.36
sudo apt update
sudo apt install mysql-server
sudo mysql_secure_installation
```

### 2. 获取测试网ETH

- Sepolia Faucet: https://sepoliafaucet.com/
- Base Sepolia Faucet: https://www.coinbase.com/faucets/base-ethereum-sepolia-faucet

## 智能合约部署

### 1. 配置环境变量

```bash
cd contracts
cp .env.example .env
```

编辑 `.env` 文件：

```env
SEPOLIA_RPC_URL=https://rpc.sepolia.org
BASE_SEPOLIA_RPC_URL=https://sepolia.base.org
PRIVATE_KEY=你的私钥（不含0x前缀）
ETHERSCAN_API_KEY=你的Etherscan API密钥
BASESCAN_API_KEY=你的Basescan API密钥
```

### 2. 安装依赖并编译

```bash
npm install
npm run compile
```

### 3. 部署合约

```bash
# 部署到Sepolia
npm run deploy:sepolia

# 部署到Base Sepolia
npm run deploy:base-sepolia
```

部署成功后，记录输出的合约地址。

### 4. 更新配置文件

编辑 `backend/config/config.yaml`，更新合约地址：

```yaml
chains:
  - id: sepolia
    contract_address: "0x你的Sepolia合约地址"
    
  - id: base-sepolia
    contract_address: "0x你的Base Sepolia合约地址"
```

## 数据库配置

### 1. 创建数据库

```bash
mysql -u root -p < database/schema.sql
```

### 2. 创建数据库用户（可选）

```sql
CREATE USER 'tokenpoints'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON token_points_system.* TO 'tokenpoints'@'localhost';
FLUSH PRIVILEGES;
```

### 3. 更新后端配置

编辑 `backend/config/config.yaml`：

```yaml
database:
  host: localhost
  port: 3306
  user: tokenpoints
  password: your_password
  dbname: token_points_system
```

## 后端服务部署

### 1. 下载依赖

```bash
cd backend
go mod download
```

### 2. 本地运行（开发环境）

```bash
go run cmd/main.go
```

### 3. 构建生产版本

```bash
go build -o token-points-backend cmd/main.go
```

### 4. 使用Systemd管理（Linux）

创建服务文件 `/etc/systemd/system/token-points.service`：

```ini
[Unit]
Description=Token Points System Backend
After=network.target mysql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/token-points/backend
ExecStart=/opt/token-points/backend/token-points-backend
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable token-points
sudo systemctl start token-points
sudo systemctl status token-points
```

## 前端部署

### 1. 使用Nginx

创建Nginx配置 `/etc/nginx/sites-available/token-points`：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    root /opt/token-points/frontend;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

启用配置：

```bash
sudo ln -s /etc/nginx/sites-available/token-points /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### 2. 使用HTTPS（推荐）

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

## 测试合约功能

### 1. 使用Hardhat控制台

```bash
cd contracts
npx hardhat console --network sepolia
```

### 2. 铸造代币

```javascript
const Token = await ethers.getContractFactory("TokenPoints");
const token = await Token.attach("0x你的合约地址");

// 铸造100个代币给指定地址
await token.mint("0x接收地址", ethers.parseEther("100"));
```

### 3. 查询余额

```javascript
const balance = await token.balanceOf("0x用户地址");
console.log(ethers.formatEther(balance));
```

### 4. 转账

```javascript
await token.transfer("0x接收地址", ethers.parseEther("10"));
```

### 5. 销毁代币

```javascript
await token.burn(ethers.parseEther("5"));
```

## 监控和日志

### 1. 查看后端日志

```bash
# Systemd日志
sudo journalctl -u token-points -f

# 或直接查看日志文件
tail -f /opt/token-points/backend/logs/app.log
```

### 2. 监控服务状态

```bash
# 检查服务状态
sudo systemctl status token-points

# 检查端口占用
sudo netstat -tulpn | grep 8080

# 检查进程
ps aux | grep token-points
```

## 故障排查

### 1. 数据库连接失败

```bash
# 检查MySQL状态
sudo systemctl status mysql

# 测试连接
mysql -u tokenpoints -p token_points_system
```

### 2. RPC连接失败

- 检查网络连接
- 尝试使用其他RPC节点
- 检查API密钥是否有效

### 3. 合约事件未捕获

- 确认合约地址正确
- 检查起始区块号设置
- 查看后端日志错误信息

### 4. 积分计算错误

- 检查余额变动记录是否完整
- 验证积分计算公式
- 使用回溯计算功能重新计算

## 性能优化

### 1. 数据库优化

```sql
-- 添加索引
CREATE INDEX idx_balance_history_composite 
ON balance_history(chain_id, user_address, timestamp);

-- 定期清理旧数据
DELETE FROM balance_history 
WHERE timestamp < DATE_SUB(NOW(), INTERVAL 90 DAY);
```

### 2. 后端优化

- 调整数据库连接池大小
- 增加区块拉取间隔
- 使用Redis缓存热点数据

### 3. 前端优化

- 启用Gzip压缩
- 使用CDN加速静态资源
- 实现分页查询

## 备份策略

### 1. 数据库备份

```bash
# 每日备份脚本
#!/bin/bash
DATE=$(date +%Y%m%d)
mysqldump -u tokenpoints -p'password' token_points_system > /backup/db_$DATE.sql
find /backup -name "db_*.sql" -mtime +30 -delete
```

### 2. 配置备份

定期备份以下文件：
- `backend/config/config.yaml`
- `contracts/.env`
- `contracts/deployments/`

## 安全建议

1. **私钥管理**：使用环境变量或密钥管理服务
2. **数据库安全**：使用强密码，限制访问IP
3. **API安全**：实现速率限制和认证
4. **HTTPS**：生产环境必须使用HTTPS
5. **定期更新**：及时更新依赖包修复安全漏洞

## 升级指南

### 1. 合约升级

- 部署新合约
- 更新配置文件中的合约地址
- 重启后端服务

### 2. 后端升级

```bash
# 拉取最新代码
git pull

# 重新构建
cd backend
go build -o token-points-backend cmd/main.go

# 重启服务
sudo systemctl restart token-points
```

### 3. 数据库迁移

```bash
# 执行迁移脚本
mysql -u tokenpoints -p token_points_system < database/migrations/xxx.sql
```
