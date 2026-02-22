package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kjannette/trahn-backend/internal/models"
)

type TradeRepo struct {
	pool *pgxpool.Pool
}

func NewTradeRepo(pool *pgxpool.Pool) *TradeRepo {
	return &TradeRepo{pool: pool}
}

func (r *TradeRepo) Record(ctx context.Context, t *models.Trade) (*models.Trade, error) {
	ts := t.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	td := TradingDay(ts)

	row := r.pool.QueryRow(ctx,
		`INSERT INTO trade_history
		 (timestamp, trading_day, side, price, quantity, usd_value,
		  grid_level, tx_hash, is_paper_trade, slippage_percent, gas_cost_eth)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING *`,
		ts, td, t.Side, t.Price, t.Quantity, t.USDValue,
		t.GridLevel, t.TxHash, t.IsPaperTrade, t.SlippagePercent, t.GasCostETH,
	)
	return scanTrade(row)
}

// GetByDay returns trades for a given trading day.
// If paperMode is non-nil, filters by is_paper_trade.
func (r *TradeRepo) GetByDay(ctx context.Context, tradingDay string, paperMode *bool) ([]models.Trade, error) {
	query, args := buildFilteredQuery(
		`SELECT * FROM trade_history WHERE trading_day = $1`,
		[]any{tradingDay},
		paperMode,
	)
	query += " ORDER BY timestamp ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTrades(rows)
}

// GetAll returns the most recent trades.
// If paperMode is non-nil, filters by is_paper_trade.
func (r *TradeRepo) GetAll(ctx context.Context, limit int, paperMode *bool) ([]models.Trade, error) {
	query, args := buildFilteredQuery(
		`SELECT * FROM trade_history WHERE 1=1`,
		nil,
		paperMode,
	)
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTrades(rows)
}

// GetStats returns aggregate trade statistics.
// If paperMode is non-nil, filters by is_paper_trade.
func (r *TradeRepo) GetStats(ctx context.Context, paperMode *bool) (*models.TradeStats, error) {
	query, args := buildFilteredQuery(
		`SELECT
			COUNT(*),
			COUNT(CASE WHEN side = 'buy' THEN 1 END),
			COUNT(CASE WHEN side = 'sell' THEN 1 END),
			SUM(usd_value),
			AVG(price),
			MIN(timestamp),
			MAX(timestamp)
		 FROM trade_history WHERE 1=1`,
		nil,
		paperMode,
	)

	var s models.TradeStats
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&s.TotalTrades, &s.BuyCount, &s.SellCount,
		&s.TotalVolume, &s.AvgPrice, &s.FirstTrade, &s.LastTrade,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *TradeRepo) CountToday(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM trade_history WHERE trading_day = $1`,
		TradingDayNow(),
	).Scan(&count)
	return count, err
}

// buildFilteredQuery appends an is_paper_trade clause when paperMode is non-nil.
func buildFilteredQuery(baseQuery string, baseArgs []any, paperMode *bool) (string, []any) {
	if paperMode == nil {
		return baseQuery, baseArgs
	}
	args := append(baseArgs, *paperMode)
	return baseQuery + fmt.Sprintf(" AND is_paper_trade = $%d", len(args)), args
}

// --- scan helpers ---

func scanTrade(row scannable) (*models.Trade, error) {
	var t models.Trade
	var td time.Time
	err := row.Scan(
		&t.ID, &t.Timestamp, &td, &t.Side, &t.Price, &t.Quantity, &t.USDValue,
		&t.GridLevel, &t.TxHash, &t.IsPaperTrade, &t.SlippagePercent, &t.GasCostETH,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.TradingDay = td.Format("2006-01-02")
	return &t, nil
}

func collectTrades(rows rowsIter) ([]models.Trade, error) {
	var out []models.Trade
	for rows.Next() {
		var t models.Trade
		var td time.Time
		if err := rows.Scan(
			&t.ID, &t.Timestamp, &td, &t.Side, &t.Price, &t.Quantity, &t.USDValue,
			&t.GridLevel, &t.TxHash, &t.IsPaperTrade, &t.SlippagePercent, &t.GasCostETH,
			&t.CreatedAt,
		); err != nil {
			return nil, err
		}
		t.TradingDay = td.Format("2006-01-02")
		out = append(out, t)
	}
	return out, rows.Err()
}
