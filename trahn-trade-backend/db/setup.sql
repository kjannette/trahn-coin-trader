-- Setup script for Trahn Grid Trader database
-- Run this as PostgreSQL superuser or database owner

-- Create database (if it doesn't exist)
-- Run manually: CREATE DATABASE trahn_grid_trader;

-- Connect to the database and run schema.sql
-- psql -U postgres -d trahn_grid_trader -f schema.sql

-- Or run this combined script:
-- psql -U postgres -f setup.sql

CREATE DATABASE IF NOT EXISTS trahn_grid_trader;
\c trahn_grid_trader;

-- Now load the schema
\i schema.sql

