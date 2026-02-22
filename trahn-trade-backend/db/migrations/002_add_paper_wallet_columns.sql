-- Migration: Add Paper Wallet Columns to grid_state
-- This allows grid_state to store paper trading virtual wallet data

ALTER TABLE grid_state 
  ADD COLUMN IF NOT EXISTS paper_eth_balance DECIMAL(18, 8),
  ADD COLUMN IF NOT EXISTS paper_usdc_balance DECIMAL(12, 2),
  ADD COLUMN IF NOT EXISTS paper_total_gas_spent DECIMAL(18, 8),
  ADD COLUMN IF NOT EXISTS paper_trades_json JSONB,
  ADD COLUMN IF NOT EXISTS paper_start_time TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS paper_initial_eth DECIMAL(18, 8),
  ADD COLUMN IF NOT EXISTS paper_initial_usdc DECIMAL(12, 2);

-- For live trading mode, these columns will be NULL
-- For paper mode, these columns store the virtual wallet state

