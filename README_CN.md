# CheetahExchange

基于 Go 构建的高性能现货交易交易所，采用 Kafka 驱动的撮合引擎、实时 WebSocket 推送和 RESTful API。

Fork 自 [gitbitex-spot](https://github.com/gitbitex/gitbitex-spot)。

## 系统架构

CheetahExchange 由三个可独立部署的服务组成：

| 服务 | 入口 | 说明 |
|------|------|------|
| **spot-core** | `cmd/spot-core` | 撮合引擎 + 后台 Worker（成交执行器、账单执行器） |
| **spot-rest** | `cmd/spot-rest` | RESTful API 服务器 |
| **spot-pushing** | `cmd/spot-pushing` | WebSocket 实时推送服务器 |

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  spot-rest   │     │  spot-core   │     │ spot-pushing │
│  (REST API)  │────▶│  (撮合引擎)   │────▶│  (WebSocket) │
│  :8001       │     │              │     │  :8002       │
└──────────────┘     └──────────────┘     └──────────────┘
       │                    │                     │
       ▼                    ▼                     ▼
  ┌─────────┐        ┌─────────┐          ┌─────────┐
  │  MySQL  │        │  Kafka  │          │  Redis  │
  └─────────┘        └─────────┘          └─────────┘
```

### 数据流

1. **下单**：客户端 → REST API → MySQL 持久化 → Kafka 发布订单消息
2. **撮合**：撮合引擎从 Kafka 读取订单 → 与订单簿撮合 → 产生成交日志写入 Kafka
3. **结算**：成交执行器和账单执行器消费成交日志 → 更新账户余额和订单状态到 MySQL
4. **实时推送**：推送服务从 Kafka/Redis 读取数据 → 通过 WebSocket 推送给客户端

## 功能特性

- **撮合引擎**：基于价格-时间优先的订单簿，支持通过 Redis 快照/恢复
- **订单类型**：限价单（Limit）、市价单（Market）
- **有效期策略**：GTC（撤销前有效）、IOC（立即成交或取消）、GTX（只做挂单）、FOK（全部成交或取消）
- **实时数据**：基于 WebSocket 的订单簿、成交和行情推送
- **账户系统**：多币种账户，支持冻结/可用余额模型
- **费率管理**：按用户等级配置 Maker/Taker 手续费率
- **钱包**：充值地址生成、提现支持
- **安全**：JWT 认证、bcrypt 密码哈希、参数化 SQL 查询

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.25 |
| Web 框架 | Gin |
| 数据库 | MySQL（Binlog ROW 格式） |
| 消息队列 | Apache Kafka |
| 缓存 | Redis |
| WebSocket | gorilla/websocket |
| ORM | GORM |
| 认证 | JWT (golang-jwt/jwt/v5) |
| 高精度数值 | shopspring/decimal |

## 环境要求

### 基础设施

- **MySQL**（需启用 BINLOG ROW 格式）
- **Apache Kafka**
- **Redis**
- **Zookeeper**（Kafka 依赖）

### Go 编译器

- [Go 1.25+](https://go.dev/doc/install)

## 安装部署

### 1. 安装基础设施

**MySQL**

```bash
sudo apt-get install mysql-server
```

启用 Binlog ROW 格式：

```ini
# /etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
server-id=1
log-bin = mysql-bin
```

MySQL 8.x 需设置 sql-mode：

```ini
# /etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
sql-mode=ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION
```

**Zookeeper**

```bash
bash bin/zkServer.sh start
```

**Kafka**

```bash
bash bin/kafka-server-start.sh config/server.properties
```

**Redis**

```bash
sudo apt-get install redis-server
```

### 2. 初始化数据库

```bash
mysql -u root -p -e "CREATE DATABASE spot CHARACTER SET utf8;"
mysql -u root -p spot < ddl.sql
mysql -u root -p spot < ddl_order.sql
```

### 3. 编译

```bash
git clone https://github.com/CheetahExchange/CheetahExchange
cd CheetahExchange
make clean
make
```

### 4. 配置

```bash
cp conf_example.json conf.json
```

编辑 `conf.json`，填入你的基础设施配置：

```json
{
  "dataSource": {
    "driverName": "mysql",
    "addr": "127.0.0.1:3306",
    "database": "spot",
    "user": "root",
    "password": "",
    "enableAutoMigrate": false
  },
  "redis": {
    "addr": ":6379",
    "password": ""
  },
  "kafka": {
    "brokers": ["localhost:9092"]
  },
  "pushServer": {
    "addr": ":8002",
    "path": "/ws"
  },
  "restServer": {
    "addr": ":8001"
  },
  "jwtSecret": "请替换为随机生成的密钥"
}
```

> **重要**：部署前务必将 `jwtSecret` 替换为加密随机字符串。

### 5. 启动服务

```bash
./spot-core
./spot-rest
./spot-pushing
```

## REST API

### 公开接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/configs` | 获取交易所配置 |
| POST | `/api/users` | 注册新用户 |
| POST | `/api/users/accessToken` | 登录（返回 JWT Token） |
| POST | `/api/users/token` | 获取访问令牌 |
| GET | `/api/products` | 获取所有交易对 |
| GET | `/api/products/:productId/trades` | 获取最近成交 |
| GET | `/api/products/:productId/book` | 获取订单簿 |
| GET | `/api/products/:productId/candles` | 获取K线数据 |

### 私有接口（需要认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/orders` | 查询用户订单 |
| POST | `/api/orders` | 下单 |
| DELETE | `/api/orders/:orderId` | 撤销订单 |
| DELETE | `/api/orders` | 批量撤单 |
| GET | `/api/accounts` | 查询账户余额 |
| GET | `/api/users/self` | 获取当前用户信息 |
| POST | `/api/users/password` | 修改密码 |
| DELETE | `/api/users/accessToken` | 登出 |
| GET | `/api/wallets/:currency/address` | 获取充值地址 |
| GET | `/api/wallets/:currency/transactions` | 查询钱包交易记录 |
| POST | `/api/wallets/:currency/withdrawal` | 提交提现请求 |

### 认证方式

所有私有接口需要通过 `accessToken` Cookie 传递有效的 JWT Token。

## WebSocket API

连接 `ws://localhost:8002/ws` 可接收实时市场数据推送。

## 项目结构

```
CheetahExchange/
├── cmd/                    # 服务入口
│   ├── spot-core/          # 撮合引擎 + Worker
│   ├── spot-rest/          # REST API 服务器
│   └── spot-pushing/       # WebSocket 推送服务器
├── conf/                   # 配置加载
├── matching/               # 撮合引擎核心
│   ├── engine.go           # 引擎主循环（拉取 → 撮合 → 提交 → 快照）
│   ├── order_book.go       # 价格-时间优先订单簿
│   ├── kafka_order_reader.go   # 从 Kafka 读取订单
│   ├── kafka_log_store.go      # 将成交日志写入 Kafka
│   └── redis_snapshot_store.go # Redis 快照存储
├── models/                 # 数据模型与持久化
│   ├── models.go           # 核心领域模型
│   ├── mysql/              # MySQL 存储实现
│   └── binlog_stream.go    # MySQL Binlog 监听
├── pushing/                # WebSocket 推送服务
│   ├── server.go           # WebSocket 服务器
│   ├── client.go           # 客户端连接处理
│   └── subscription.go     # 频道订阅管理
├── rest/                   # REST API 层
│   ├── server.go           # 路由定义
│   ├── auth.go             # JWT 认证中间件
│   └── *_controller.go     # 请求处理器
├── service/                # 业务逻辑层
├── utils/                  # 工具函数
├── worker/                 # 后台 Worker
├── ddl.sql                 # 数据库建表语句（账户、交易对、成交等）
├── ddl_order.sql           # 分片订单表建表语句
└── conf_example.json       # 配置模板
```

## 数据库

| 表名 | 说明 |
|------|------|
| `g_user` | 用户账户 |
| `g_account` | 多币种余额（冻结 + 可用） |
| `g_bill` | 账户变动记录 |
| `g_product` | 交易对（如 BTC-USDT） |
| `g_order_0` ~ `g_order_127` | 分片订单表 |
| `g_fill` | 成交记录 |
| `g_trade` | 已执行交易 |
| `g_tick` | K线数据（OHLCV） |
| `g_fee_rate` | 用户等级费率 |
| `g_config` | 系统配置 |

订单表按 128 个分片（`g_order_0` 至 `g_order_127`）拆分，提升写入性能。

## 许可证

本项目遵循 [LICENSE](LICENSE) 文件中的许可条款。