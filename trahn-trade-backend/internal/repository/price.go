package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kjannette/trahn-backend/internal/models"
)

type PriceRepo struct {
	pool *pgxpool.Pool
}

func NewPriceRepo(pool *pgxpool.Pool) *PriceRepo {
	return &PriceRepo{pool: pool}
}

func (r *PriceRepo) Record(ctx context.Context, price float64, ts time.Time) (*models.PricePoint, error) {
	td := TradingDay(ts)
	row := r.pool.QueryRow(ctx,
		`INSERT INTO price_history (timestamp, price, trading_day, source)
		 VALUES ($1, $2, $3, $4) RETURNING *`,
		ts, price, td, "coingecko",
	)
	return scanPrice(row)
}

func (r *PriceRepo) GetByDay(ctx context.Context, tradingDay string) ([]models.PricePoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT * FROM price_history WHERE trading_day = $1 ORDER BY timestamp ASC`,
		tradingDay,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPrices(rows)
}

func (r *PriceRepo) GetAvailableDays(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT trading_day FROM price_history ORDER BY trading_day ASC LIMIT 30`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var days []string
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		days = append(days, d.Format("2006-01-02"))
	}
	return days, rows.Err()
}

func (r *PriceRepo) GetLatest(ctx context.Context) (*models.PricePoint, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT * FROM price_history ORDER BY timestamp DESC LIMIT 1`,
	)
	p, err := scanPrice(row)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

// --- scan helpers ---

type scannable interface {
	Scan(dest ...any) error
}

func scanPrice(row scannable) (*models.PricePoint, error) {
	var p models.PricePoint
	var td time.Time
	err := row.Scan(&p.ID, &p.Timestamp, &p.Price, &td, &p.Source, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.TradingDay = td.Format("2006-01-02")
	return &p, nil
}

type rowsIter interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func collectPrices(rows rowsIter) ([]models.PricePoint, error) {
	var out []models.PricePoint
	for rows.Next() {
		var p models.PricePoint
		var td time.Time
		if err := rows.Scan(&p.ID, &p.Timestamp, &p.Price, &td, &p.Source, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.TradingDay = td.Format("2006-01-02")
		out = append(out, p)
	}
	return out, rows.Err()
}
