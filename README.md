# CheetahExchange

A high-performance spot trading exchange built with Go, featuring a Kafka-driven matching engine, real-time WebSocket pushing, and a RESTful API.

Forked from [gitbitex-spot](https://github.com/gitbitex/gitbitex-spot).

## Architecture

CheetahExchange consists of three independently deployable services:

| Service | Entry Point | Description |
|---------|-------------|-------------|
| **spot-core** | `cmd/spot-core` | Matching engine + workers (fill executor, bill executor) |
| **spot-rest** | `cmd/spot-rest` | RESTful API server |
| **spot-pushing** | `cmd/spot-pushing` | WebSocket push server for real-time market data |

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  spot-rest   │     │  spot-core   │     │ spot-pushing │
│  (REST API)  │────▶│  (Matching   │────▶│  (WebSocket) │
│  :8001       │     │   Engine)    │     │  :8002       │
└──────────────┘     └──────────────┘     └──────────────┘
       │                    │                     │
       ▼                    ▼                     ▼
  ┌─────────┐        ┌─────────┐          ┌─────────┐
  │  MySQL  │        │  Kafka  │          │  Redis  │
  └─────────┘        └─────────┘          └─────────┘
```

### Data Flow

1. **Order Placement**: Client → REST API → MySQL → Kafka
2. **Order Matching**: Matching engine reads from Kafka → matches against order book → produces trade logs to Kafka
3. **Settlement**: Fill/Bill executors consume trade logs → update account balances and order status in MySQL
4. **Real-time Push**: Pushing service reads from Kafka/Redis → pushes updates to WebSocket clients

## Features

- **Matching Engine**: Price-time priority order book with snapshot/recovery via Redis
- **Order Types**: Limit, Market
- **Time in Force**: GTC (Good Till Canceled), IOC (Immediate Or Cancel), GTX (Good Till Crossing), FOK (Fill Or Kill)
- **Real-time Data**: WebSocket-based order book, trades, and ticker streaming
- **Account System**: Multi-currency accounts with hold/available balance model
- **Fee Schedule**: Configurable maker/taker fee rates by user level
- **Wallet**: Deposit address generation, withdrawal support
- **Security**: JWT authentication, bcrypt password hashing, parameterized SQL queries

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25 |
| Web Framework | Gin |
| Database | MySQL (Binlog ROW format) |
| Message Queue | Apache Kafka |
| Cache | Redis |
| WebSocket | gorilla/websocket |
| ORM | GORM |
| Authentication | JWT (golang-jwt/jwt/v5) |
| Precise Decimals | shopspring/decimal |

## Prerequisites

### Infrastructure

- **MySQL** (with BINLOG ROW format enabled)
- **Apache Kafka**
- **Redis**
- **Zookeeper** (required by Kafka)

### Go Compiler

- [Go 1.25+](https://go.dev/doc/install)

## Installation

### 1. Set Up Infrastructure

**MySQL**

```bash
sudo apt-get install mysql-server
```

Enable binlog in ROW format:

```ini
# /etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
server-id=1
log-bin = mysql-bin
```

For MySQL 8.x, set sql-mode:

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

### 2. Initialize Database

```bash
mysql -u root -p -e "CREATE DATABASE spot CHARACTER SET utf8;"
mysql -u root -p spot < ddl.sql
mysql -u root -p spot < ddl_order.sql
```

### 3. Build

```bash
git clone https://github.com/CheetahExchange/CheetahExchange
cd CheetahExchange
make clean
make
```

### 4. Configure

```bash
cp conf_example.json conf.json
```

Edit `conf.json` with your infrastructure settings:

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
  "jwtSecret": "CHANGE_ME_TO_A_RANDOM_SECRET"
}
```

> **Important**: Always change `jwtSecret` to a cryptographically random string before deploying.

### 5. Run

```bash
./spot-core
./spot-rest
./spot-pushing
```

## REST API

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/configs` | Get exchange configuration |
| POST | `/api/users` | Register a new user |
| POST | `/api/users/accessToken` | Sign in (returns JWT token) |
| POST | `/api/users/token` | Get access token |
| GET | `/api/products` | List all trading products |
| GET | `/api/products/:productId/trades` | Get recent trades |
| GET | `/api/products/:productId/book` | Get order book |
| GET | `/api/products/:productId/candles` | Get candlestick data |

### Private Endpoints (Authentication Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/orders` | List user's orders |
| POST | `/api/orders` | Place a new order |
| DELETE | `/api/orders/:orderId` | Cancel an order |
| DELETE | `/api/orders` | Cancel multiple orders |
| GET | `/api/accounts` | List account balances |
| GET | `/api/users/self` | Get current user profile |
| POST | `/api/users/password` | Change password |
| DELETE | `/api/users/accessToken` | Sign out |
| GET | `/api/wallets/:currency/address` | Get deposit address |
| GET | `/api/wallets/:currency/transactions` | Get wallet transactions |
| POST | `/api/wallets/:currency/withdrawal` | Submit withdrawal |

### Authentication

All private endpoints require a valid JWT token sent via the `accessToken` cookie.

## WebSocket API

Connect to `ws://localhost:8002/ws` to receive real-time market data updates.

## Project Structure

```
CheetahExchange/
├── cmd/                    # Service entry points
│   ├── spot-core/          # Matching engine + workers
│   ├── spot-rest/          # REST API server
│   └── spot-pushing/       # WebSocket push server
├── conf/                   # Configuration loader
├── matching/               # Matching engine core
│   ├── engine.go           # Engine loop (fetch → apply → commit → snapshot)
│   ├── order_book.go       # Price-time priority order book
│   ├── kafka_order_reader.go   # Read orders from Kafka
│   ├── kafka_log_store.go      # Persist trade logs to Kafka
│   └── redis_snapshot_store.go # Snapshot storage in Redis
├── models/                 # Data models and persistence
│   ├── models.go           # Core domain models
│   ├── mysql/              # MySQL store implementations
│   └── binlog_stream.go    # MySQL binlog stream listener
├── pushing/                # WebSocket push service
│   ├── server.go           # WebSocket server
│   ├── client.go           # Client connection handler
│   └── subscription.go     # Channel subscription manager
├── rest/                   # REST API layer
│   ├── server.go           # Route definitions
│   ├── auth.go             # JWT authentication middleware
│   └── *_controller.go     # Request handlers
├── service/                # Business logic layer
├── utils/                  # Utility functions
├── worker/                 # Background workers
├── ddl.sql                 # Database schema (accounts, products, trades, etc.)
├── ddl_order.sql           # Sharded order table schema
└── conf_example.json       # Configuration template
```

## Database

| Table | Description |
|-------|-------------|
| `g_user` | User accounts |
| `g_account` | Multi-currency balances (hold + available) |
| `g_bill` | Account change records |
| `g_product` | Trading pairs (e.g., BTC-USDT) |
| `g_order_0` .. `g_order_127` | Sharded order tables |
| `g_fill` | Trade fill records |
| `g_trade` | Executed trades |
| `g_tick` | Candlestick (OHLCV) data |
| `g_fee_rate` | Fee rates by user level |
| `g_config` | System configuration |

Order tables are sharded into 128 partitions (`g_order_0` through `g_order_127`) for write scalability.

## License

This project is licensed under the terms found in the [LICENSE](LICENSE) file.