package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kjannette/trahn-backend/internal/models"
	"github.com/kjannette/trahn-backend/internal/repository"
	"github.com/kjannette/trahn-backend/internal/testutil"
)

// ---------- PriceRepo ----------

func TestPriceRepo(t *testing.T) {
	pool := testutil.SetupPool(t)
	repo := repository.NewPriceRepo(pool)
	ctx := context.Background()

	// Record
	ts := time.Now()
	p, err := repo.Record(ctx, 2650.42, ts)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if p.Price != 2650.42 {
		t.Fatalf("price mismatch: got %f", p.Price)
	}
	t.Logf("Recorded price: id=%d price=%.2f day=%s", p.ID, p.Price, p.TradingDay)

	// GetLatest
	latest, err := repo.GetLatest(ctx)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest price")
	}
	t.Logf("Latest: id=%d price=%.2f", latest.ID, latest.Price)

	// GetByDay
	prices, err := repo.GetByDay(ctx, p.TradingDay)
	if err != nil {
		t.Fatalf("GetByDay: %v", err)
	}
	if len(prices) == 0 {
		t.Fatal("expected prices for trading day")
	}
	t.Logf("GetByDay(%s): %d rows", p.TradingDay, len(prices))

	// GetAvailableDays
	days, err := repo.GetAvailableDays(ctx)
	if err != nil {
		t.Fatalf("GetAvailableDays: %v", err)
	}
	if len(days) == 0 {
		t.Fatal("expected at least one day")
	}
	t.Logf("Available days: %v", days)
}

// ---------- TradeRepo ----------

func TestTradeRepo(t *testing.T) {
	pool := testutil.SetupPool(t)
	repo := repository.NewTradeRepo(pool)
	ctx := context.Background()

	slippage := 0.35
	gasCost := 0.002
	gridLvl := 3

	trade := &models.Trade{
		Timestamp:       time.Now(),
		Side:            "buy",
		Price:           2600.00,
		Quantity:        0.0385,
		USDValue:        100.00,
		GridLevel:       &gridLvl,
		IsPaperTrade:    true,
		SlippagePercent: &slippage,
		GasCostETH:      &gasCost,
	}

	recorded, err := repo.Record(ctx, trade)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if recorded.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if recorded.Side != "buy" {
		t.Fatalf("side mismatch: got %s", recorded.Side)
	}
	t.Logf("Recorded trade: id=%d side=%s price=%.2f qty=%.4f", recorded.ID, recorded.Side, recorded.Price, recorded.Quantity)

	// GetAll (no filter)
	all, err := repo.GetAll(ctx, 10, nil)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("expected trades")
	}
	t.Logf("GetAll: %d trades", len(all))

	// GetAll (paper only)
	paperMode := true
	paperTrades, err := repo.GetAll(ctx, 10, &paperMode)
	if err != nil {
		t.Fatalf("GetAll(paper): %v", err)
	}
	for _, pt := range paperTrades {
		if !pt.IsPaperTrade {
			t.Fatalf("expected paper trade, got live trade id=%d", pt.ID)
		}
	}
	t.Logf("GetAll(paper): %d trades", len(paperTrades))

	// GetStats (no filter)
	stats, err := repo.GetStats(ctx, nil)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	t.Logf("Stats(all): total=%d buys=%d sells=%d", stats.TotalTrades, stats.BuyCount, stats.SellCount)

	// GetStats (paper only)
	paperStats, err := repo.GetStats(ctx, &paperMode)
	if err != nil {
		t.Fatalf("GetStats(paper): %v", err)
	}
	t.Logf("Stats(paper): total=%d buys=%d sells=%d", paperStats.TotalTrades, paperStats.BuyCount, paperStats.SellCount)
}

// ---------- SRRepo ----------

func TestSRRepo(t *testing.T) {
	pool := testutil.SetupPool(t)
	repo := repository.NewSRRepo(pool)
	ctx := context.Background()

	avgPrice := 2700.0
	sr := &models.SupportResistance{
		Timestamp:        time.Now(),
		Method:           "simple",
		LookbackDays:     14,
		Support:          2400.00,
		Resistance:       3000.00,
		Midpoint:         2700.00,
		AvgPrice:         &avgPrice,
		GridRecalculated: false,
	}

	recorded, err := repo.Record(ctx, sr)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if recorded.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	t.Logf("Recorded S/R: id=%d support=%.2f resistance=%.2f mid=%.2f", recorded.ID, recorded.Support, recorded.Resistance, recorded.Midpoint)

	// GetLatest
	latest, err := repo.GetLatest(ctx)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest S/R")
	}
	t.Logf("Latest S/R: mid=%.2f", latest.Midpoint)

	// GetHistory
	history, err := repo.GetHistory(ctx, 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	t.Logf("S/R history: %d rows", len(history))

	// NeedsRefresh (just recorded, so should NOT need refresh)
	needs, err := repo.NeedsRefresh(ctx, 48)
	if err != nil {
		t.Fatalf("NeedsRefresh: %v", err)
	}
	if needs {
		t.Fatal("should NOT need refresh right after recording")
	}
	t.Logf("NeedsRefresh(48h): %v", needs)

	// CheckSignificantChange
	shifted := &models.SupportResistance{Midpoint: 2900.00}
	analysis, err := repo.CheckSignificantChange(ctx, shifted, 5)
	if err != nil {
		t.Fatalf("CheckSignificantChange: %v", err)
	}
	t.Logf("Change analysis: changed=%v reason=%s", analysis.HasChanged, analysis.Reason)
}

// ---------- GridStateRepo ----------

func TestGridStateRepo(t *testing.T) {
	pool := testutil.SetupPool(t)
	repo := repository.NewGridStateRepo(pool)
	ctx := context.Background()

	basePrice := 2700.0
	levels := json.RawMessage(`[{"index":0,"price":2600,"side":"buy","filled":false}]`)

	gs := &models.GridState{
		BasePrice:      &basePrice,
		GridLevelsJSON: levels,
		TradesExecuted: 0,
		TotalProfit:    0,
	}

	saved, err := repo.Save(ctx, gs)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if !saved.IsActive {
		t.Fatal("expected active state")
	}
	t.Logf("Saved grid state: id=%d active=%v", saved.ID, saved.IsActive)

	// GetActive
	active, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if active == nil {
		t.Fatal("expected active state")
	}
	t.Logf("Active grid state: id=%d", active.ID)

	// UpdateGridLevels
	newLevels := json.RawMessage(`[{"index":0,"price":2600,"side":"buy","filled":true}]`)
	if err := repo.UpdateGridLevels(ctx, newLevels); err != nil {
		t.Fatalf("UpdateGridLevels: %v", err)
	}
	t.Log("UpdateGridLevels: OK")

	// UpdateTradeStats
	if err := repo.UpdateTradeStats(ctx, 5, 12.50); err != nil {
		t.Fatalf("UpdateTradeStats: %v", err)
	}
	t.Log("UpdateTradeStats: OK")

	// InitializePaperWallet
	if err := repo.InitializePaperWallet(ctx, 1.0, 1000.0); err != nil {
		t.Fatalf("InitializePaperWallet: %v", err)
	}
	t.Log("InitializePaperWallet: OK")

	// GetPaperWallet
	pw, err := repo.GetPaperWallet(ctx)
	if err != nil {
		t.Fatalf("GetPaperWallet: %v", err)
	}
	if pw == nil {
		t.Fatal("expected paper wallet")
	}
	if pw.ETHBalance != 1.0 {
		t.Fatalf("ETH balance mismatch: got %f", pw.ETHBalance)
	}
	t.Logf("PaperWallet: ETH=%.4f USDC=%.2f", pw.ETHBalance, pw.USDCBalance)

	// UpdatePaperWallet
	pw.ETHBalance = 1.05
	pw.USDCBalance = 870.00
	pw.TotalGasSpent = 0.003
	if err := repo.UpdatePaperWallet(ctx, pw); err != nil {
		t.Fatalf("UpdatePaperWallet: %v", err)
	}
	t.Log("UpdatePaperWallet: OK")

	// Verify update
	pw2, err := repo.GetPaperWallet(ctx)
	if err != nil {
		t.Fatalf("GetPaperWallet after update: %v", err)
	}
	if pw2.ETHBalance != 1.05 {
		t.Fatalf("ETH balance mismatch after update: got %f", pw2.ETHBalance)
	}
	t.Logf("PaperWallet after update: ETH=%.4f USDC=%.2f gas=%.4f", pw2.ETHBalance, pw2.USDCBalance, pw2.TotalGasSpent)

	// GetHistory
	history, err := repo.GetHistory(ctx, 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	t.Logf("Grid state history: %d rows", len(history))
}

// ---------- TradingDay ----------

func TestTradingDay(t *testing.T) {
	// 2024-01-15 at 16:00 UTC (before 17:00 cutoff) => trading day = Jan 14
	ts := time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC)
	got := repository.TradingDay(ts)
	if got != "2024-01-14" {
		t.Fatalf("expected 2024-01-14, got %s", got)
	}

	// 2024-01-15 at 18:00 UTC (after 17:00 cutoff) => trading day = Jan 15
	ts2 := time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC)
	got2 := repository.TradingDay(ts2)
	if got2 != "2024-01-15" {
		t.Fatalf("expected 2024-01-15, got %s", got2)
	}

	t.Logf("TradingDay tests passed")
}
