package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kjannette/trahn-backend/internal/models"
)

type SRRepo struct {
	pool *pgxpool.Pool
}

func NewSRRepo(pool *pgxpool.Pool) *SRRepo {
	return &SRRepo{pool: pool}
}

func (r *SRRepo) Record(ctx context.Context, sr *models.SupportResistance) (*models.SupportResistance, error) {
	ts := sr.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO support_resistance_history
		 (timestamp, method, lookback_days, support, resistance, midpoint, avg_price, grid_recalculated)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING *`,
		ts, sr.Method, sr.LookbackDays, sr.Support, sr.Resistance,
		sr.Midpoint, sr.AvgPrice, sr.GridRecalculated,
	)
	return scanSR(row)
}

func (r *SRRepo) GetLatest(ctx context.Context) (*models.SupportResistance, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT * FROM support_resistance_history ORDER BY timestamp DESC LIMIT 1`,
	)
	sr, err := scanSR(row)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return sr, nil
}

func (r *SRRepo) GetHistory(ctx context.Context, limit int) ([]models.SupportResistance, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT * FROM support_resistance_history ORDER BY timestamp DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSRs(rows)
}

func (r *SRRepo) NeedsRefresh(ctx context.Context, refreshHours int) (bool, error) {
	latest, err := r.GetLatest(ctx)
	if err != nil {
		return false, err
	}
	if latest == nil {
		return true, nil
	}
	age := time.Since(latest.Timestamp)
	return age >= time.Duration(refreshHours)*time.Hour, nil
}

type ChangeAnalysis struct {
	HasChanged    bool                       `json:"hasChanged"`
	ChangePercent *float64                   `json:"changePercent"`
	Previous      *models.SupportResistance  `json:"previous"`
	Reason        string                     `json:"reason"`
}

func (r *SRRepo) CheckSignificantChange(ctx context.Context, newSR *models.SupportResistance, thresholdPercent float64) (*ChangeAnalysis, error) {
	previous, err := r.GetLatest(ctx)
	if err != nil {
		return nil, err
	}
	if previous == nil {
		return &ChangeAnalysis{
			HasChanged: true,
			Reason:     "First S/R fetch",
		}, nil
	}

	pct := math.Abs((newSR.Midpoint - previous.Midpoint) / previous.Midpoint * 100)
	changed := pct >= thresholdPercent

	reason := "S/R stable"
	if changed {
		reason = fmt.Sprintf("Midpoint changed %.2f%%", pct)
	}

	return &ChangeAnalysis{
		HasChanged:    changed,
		ChangePercent: &pct,
		Previous:      previous,
		Reason:        reason,
	}, nil
}

// --- scan helpers ---

func scanSR(row scannable) (*models.SupportResistance, error) {
	var sr models.SupportResistance
	err := row.Scan(
		&sr.ID, &sr.Timestamp, &sr.Method, &sr.LookbackDays,
		&sr.Support, &sr.Resistance, &sr.Midpoint, &sr.AvgPrice,
		&sr.GridRecalculated, &sr.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sr, nil
}

func collectSRs(rows rowsIter) ([]models.SupportResistance, error) {
	var out []models.SupportResistance
	for rows.Next() {
		var sr models.SupportResistance
		if err := rows.Scan(
			&sr.ID, &sr.Timestamp, &sr.Method, &sr.LookbackDays,
			&sr.Support, &sr.Resistance, &sr.Midpoint, &sr.AvgPrice,
			&sr.GridRecalculated, &sr.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	return out, rows.Err()
}
