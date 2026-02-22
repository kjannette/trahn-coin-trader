# Database Setup

## Quick Start

### 1. Install And Run PostgreSQL

**macOS:**
```bash
brew install postgresql@15
brew services start postgresql@15
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

**Docker:**
```bash
docker run --name trahn-postgres \
  -e POSTGRES_PASSWORD=your_password \
  -p 5432:5432 \
  -d postgres:15
```

### 2. Create Database and Schema

```bash
# Connect as postgres user
psql -U postgres

# Create database
CREATE DATABASE trahn_grid_trader;

# Exit psql
\q

# Run schema
psql -U postgres -d trahn_grid_trader -f db/schema.sql
```

### 3. Configure .env

Add to your `.env` file:
```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=trahn_grid_trader
DB_USER=postgres
DB_PASSWORD=your_secure_password_here
```

### 4. Test Connection

```bash
make dev
# Verify "[DB] Connection successful" appears in the output
```

## Schema Overview

### Tables

1. **price_history** - ETH price data points
   - Timestamp-indexed for fast queries
   - Organized by trading day (12:00 EST boundary)

2. **trade_history** - Executed trades
   - Buy/sell records with full details
   - Paper trade flag for simulation tracking

3. **support_resistance_history** - S/R levels over time
   - Historical record of Dune API fetches
   - Tracks when grid was recalculated

4. **grid_state** - Current grid configuration
   - Grid levels stored as JSONB
   - State tracking for bot restarts

### Indexes

- All tables indexed on `timestamp` for time-series queries
- `trading_day` indexed for day-based queries
- Optimized for append-heavy workloads

## Maintenance

### View Data

```sql
-- Recent prices
SELECT * FROM price_history ORDER BY timestamp DESC LIMIT 10;

-- Recent trades
SELECT * FROM trade_history ORDER BY timestamp DESC LIMIT 10;

-- S/R history
SELECT * FROM support_resistance_history ORDER BY timestamp DESC LIMIT 10;

-- Latest S/R (using view)
SELECT * FROM latest_support_resistance;
```

### Cleanup Old Data (optional)

```sql
-- Delete price data older than 90 days
DELETE FROM price_history WHERE timestamp < NOW() - INTERVAL '90 days';

-- Keep all trade history (don't delete)
```

## Troubleshooting

### Connection Failed
- Check PostgreSQL is running: `pg_isready`
- Verify credentials in `.env`
- Check firewall/network settings

### Schema Errors
- Drop and recreate: `DROP DATABASE trahn_grid_trader; CREATE DATABASE trahn_grid_trader;`
- Re-run schema.sql

### Performance
- If queries slow, add more indexes
- Consider partitioning price_history by month (for large datasets)

