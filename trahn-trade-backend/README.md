# 📊 Trahn Grid Trader

A sophisticated grid trading bot for Uniswap V2, designed to profit from price oscillations in cryptocurrency markets.

```
================================================================
||                                                            ||
||                   7   R   4   H   N                        ||
||                                                            ||
||              G R I D   T R A D E R   v0.2                  ||
================================================================
```

## Grid Trading

An algorthmic trading strategy that places buy and sell orders at predetermined price intervals (a "grid"). The algorthm raalizes gains over cost basis from price oscillations within the grid range.

- **When price drops**: Buy orders are triggered at lower grid levels
- **When price rises**: Sell orders are triggered at higher grid levels
- **Profit**: Is derved from the spread 

```
Price
  │
  │  ────────  SELL Level 5  ($0.035)
  │  ────────  SELL Level 4  ($0.034)
  │  ────────  SELL Level 3  ($0.033)
  │  ════════  CENTER PRICE  ($0.032)  ← Grid initialized here
  │  ────────  BUY Level 2   ($0.031)
  │  ────────  BUY Level 1   ($0.030)
  │  ────────  BUY Level 0   ($0.029)
  │
  └─────────────────────────────────────►
```

## Quick Start

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- PostgreSQL 15+

### Setup

1. Clone the repository and install dependencies:

```bash
git clone https://github.com/kjannette/trahn-backend.git
cd trahn-trade-backend
go mod download
```

2. Create a `.env` file in the project root with your configuration:

```bash
# Required
WALLET_ADDRESS=0xYourWalletAddress

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=trahn_grid_trader
DB_USER=postgres
DB_PASSWORD=yourpassword

# Optional - enables support/resistance scheduling
DUNE_API_KEY=your_dune_api_key

# Paper trading is enabled by default.
# Set to false and provide PRIVATE_KEY for live trading.
PAPER_TRADING_ENABLED=true
```

3. Build and run:

```bash
go build -o trahn-bot ./cmd/server
./trahn-bot
```

Or run directly without building:

```bash
go run ./cmd/server
```

### Running Tests

```bash
go test ./...
```
