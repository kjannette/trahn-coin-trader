package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kjannette/trahn-backend/internal/models"
)

type GridStateRepo struct {
	pool *pgxpool.Pool
}

func NewGridStateRepo(pool *pgxpool.Pool) *GridStateRepo {
	return &GridStateRepo{pool: pool}
}

func (r *GridStateRepo) GetActive(ctx context.Context) (*models.GridState, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT * FROM grid_state WHERE is_active = true ORDER BY updated_at DESC LIMIT 1`,
	)
	gs, err := scanGridState(row)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return gs, nil
}

func (r *GridStateRepo) Save(ctx context.Context, data *models.GridState) (*models.GridState, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE grid_state SET is_active = false WHERE is_active = true`)
	if err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx,
		`INSERT INTO grid_state
		 (base_price, grid_levels_json, trades_executed, total_profit,
		  last_sr_refresh, is_active, updated_at)
		 VALUES ($1,$2,$3,$4,$5,true,NOW())
		 RETURNING *`,
		data.BasePrice,
		data.GridLevelsJSON,
		data.TradesExecuted,
		data.TotalProfit,
		data.LastSRRefresh,
	)
	gs, err := scanGridState(row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return gs, nil
}

func (r *GridStateRepo) UpdateGridLevels(ctx context.Context, levels json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE grid_state SET grid_levels_json = $1, updated_at = NOW() WHERE is_active = true`,
		levels,
	)
	return err
}

func (r *GridStateRepo) UpdateTradeStats(ctx context.Context, tradesExecuted int, totalProfit float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE grid_state SET trades_executed = $1, total_profit = $2, updated_at = NOW() WHERE is_active = true`,
		tradesExecuted, totalProfit,
	)
	return err
}

func (r *GridStateRepo) UpdatePaperWallet(ctx context.Context, pw *models.PaperWallet) error {
	tradesJSON, err := json.Marshal(pw.Trades)
	if err != nil {
		tradesJSON = []byte("[]")
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE grid_state
		 SET paper_eth_balance = $1,
		     paper_usdc_balance = $2,
		     paper_total_gas_spent = $3,
		     paper_trades_json = $4,
		     updated_at = NOW()
		 WHERE is_active = true`,
		pw.ETHBalance, pw.USDCBalance, pw.TotalGasSpent, tradesJSON,
	)
	return err
}

func (r *GridStateRepo) InitializePaperWallet(ctx context.Context, initialETH, initialUSDC float64) error {
	state, err := r.GetActive(ctx)
	if err != nil {
		return err
	}
	if state != nil && state.PaperETHBalance != nil {
		return nil // already initialized
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE grid_state
		 SET paper_eth_balance = $1,
		     paper_usdc_balance = $2,
		     paper_initial_eth = $1,
		     paper_initial_usdc = $2,
		     paper_total_gas_spent = 0,
		     paper_trades_json = '[]'::jsonb,
		     paper_start_time = NOW(),
		     updated_at = NOW()
		 WHERE is_active = true`,
		initialETH, initialUSDC,
	)
	return err
}

func (r *GridStateRepo) GetPaperWallet(ctx context.Context) (*models.PaperWallet, error) {
	state, err := r.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if state == nil || state.PaperETHBalance == nil {
		return nil, nil
	}
	pw := &models.PaperWallet{
		ETHBalance:    valOr(state.PaperETHBalance, 0),
		USDCBalance:   valOr(state.PaperUSDCBalance, 0),
		TotalGasSpent: valOr(state.PaperTotalGasSpent, 0),
		Trades:        state.PaperTradesJSON,
		StartTime:     state.PaperStartTime,
		InitialETH:    valOr(state.PaperInitialETH, 0),
		InitialUSDC:   valOr(state.PaperInitialUSDC, 0),
	}
	return pw, nil
}

func (r *GridStateRepo) GetHistory(ctx context.Context, limit int) ([]models.GridState, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT * FROM grid_state ORDER BY updated_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectGridStates(rows)
}

// --- scan helpers ---

func scanGridState(row scannable) (*models.GridState, error) {
	var gs models.GridState
	err := row.Scan(
		&gs.ID, &gs.BasePrice, &gs.GridLevelsJSON,
		&gs.TradesExecuted, &gs.TotalProfit, &gs.LastSRRefresh,
		&gs.IsActive, &gs.CreatedAt, &gs.UpdatedAt,
		// paper wallet columns
		&gs.PaperETHBalance, &gs.PaperUSDCBalance, &gs.PaperTotalGasSpent,
		&gs.PaperTradesJSON, &gs.PaperStartTime,
		&gs.PaperInitialETH, &gs.PaperInitialUSDC,
	)
	if err != nil {
		return nil, err
	}
	return &gs, nil
}

func collectGridStates(rows rowsIter) ([]models.GridState, error) {
	var out []models.GridState
	for rows.Next() {
		var gs models.GridState
		if err := rows.Scan(
			&gs.ID, &gs.BasePrice, &gs.GridLevelsJSON,
			&gs.TradesExecuted, &gs.TotalProfit, &gs.LastSRRefresh,
			&gs.IsActive, &gs.CreatedAt, &gs.UpdatedAt,
			&gs.PaperETHBalance, &gs.PaperUSDCBalance, &gs.PaperTotalGasSpent,
			&gs.PaperTradesJSON, &gs.PaperStartTime,
			&gs.PaperInitialETH, &gs.PaperInitialUSDC,
		); err != nil {
			return nil, err
		}
		out = append(out, gs)
	}
	return out, rows.Err()
}

func valOr(p *float64, fallback float64) float64 {
	if p != nil {
		return *p
	}
	return fallback
}
