-- Trahn Grid Trader Database Schema
-- PostgreSQL

-- Table 1: Price History
-- Stores ETH price data points from CoinGecko
CREATE TABLE IF NOT EXISTS price_history (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    price DECIMAL(12, 2) NOT NULL,
    trading_day DATE NOT NULL,
    source VARCHAR(50) DEFAULT 'coingecko',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_price_timestamp ON price_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_price_trading_day ON price_history(trading_day);

-- Table 2: Trade History
-- Stores executed trades (buys and sells)
CREATE TABLE IF NOT EXISTS trade_history (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    trading_day DATE NOT NULL,
    side VARCHAR(10) NOT NULL CHECK (side IN ('buy', 'sell')),
    price DECIMAL(12, 2) NOT NULL,
    quantity DECIMAL(18, 8) NOT NULL,
    usd_value DECIMAL(12, 2) NOT NULL,
    grid_level INTEGER,
    tx_hash VARCHAR(66),
    is_paper_trade BOOLEAN DEFAULT FALSE,
    slippage_percent DECIMAL(5, 3),
    gas_cost_eth DECIMAL(18, 8),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trade_timestamp ON trade_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_trade_trading_day ON trade_history(trading_day);
CREATE INDEX IF NOT EXISTS idx_trade_side ON trade_history(side);

-- Table 3: Support/Resistance History
-- Stores S/R levels fetched from Dune Analytics over time
CREATE TABLE IF NOT EXISTS support_resistance_history (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    method VARCHAR(20) NOT NULL,
    lookback_days INTEGER NOT NULL,
    support DECIMAL(12, 2) NOT NULL,
    resistance DECIMAL(12, 2) NOT NULL,
    midpoint DECIMAL(12, 2) NOT NULL,
    avg_price DECIMAL(12, 2),
    grid_recalculated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sr_timestamp ON support_resistance_history(timestamp);

-- Table 4: Grid State
-- Stores current grid configuration and state
-- Only one active row at a time (or keyed by bot instance)
CREATE TABLE IF NOT EXISTS grid_state (
    id SERIAL PRIMARY KEY,
    base_price DECIMAL(12, 2),
    grid_levels_json JSONB,
    trades_executed INTEGER DEFAULT 0,
    total_profit DECIMAL(12, 2) DEFAULT 0,
    last_sr_refresh TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grid_active ON grid_state(is_active);

-- View: Latest S/R
-- Convenience view for getting the most recent S/R data
CREATE OR REPLACE VIEW latest_support_resistance AS
SELECT * FROM support_resistance_history
ORDER BY timestamp DESC
LIMIT 1;

