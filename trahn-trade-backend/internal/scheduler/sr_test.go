package scheduler_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kjannette/trahn-backend/internal/external"
	"github.com/kjannette/trahn-backend/internal/repository"
	"github.com/kjannette/trahn-backend/internal/scheduler"
	"github.com/kjannette/trahn-backend/internal/strategy"
	"github.com/kjannette/trahn-backend/internal/testutil"
)

func TestSRScheduler_FetchNow(t *testing.T) {
	apiKey := os.Getenv("DUNE_API_KEY")
	if apiKey == "" {
		t.Skip("DUNE_API_KEY not set, skipping")
	}

	pool := testutil.SetupPool(t)
	srRepo := repository.NewSRRepo(pool)
	dune := external.NewDuneClient(apiKey, external.DuneOptions{
		Method:       "simple",
		LookbackDays: 14,
		RefreshHours: 48,
	})

	var srUpdated atomic.Bool
	var recalculated atomic.Bool

	sched := scheduler.NewSRScheduler(dune, srRepo, scheduler.SRSchedulerConfig{
		CronInterval:      1 * time.Hour,
		SRChangeThreshold: 5,
		OnSRUpdate: func(sr *external.SRResult) {
			srUpdated.Store(true)
		},
		OnGridRecalculate: func(sr *external.SRResult) {
			recalculated.Store(true)
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	err := sched.FetchNow(ctx)
	if err != nil {
		t.Fatalf("FetchNow: %v", err)
	}

	if !srUpdated.Load() {
		t.Fatal("OnSRUpdate callback was not called")
	}
	t.Log("OnSRUpdate: called")
	t.Logf("OnGridRecalculate called: %v", recalculated.Load())

	// Verify it was stored in DB
	latest, err := srRepo.GetLatest(ctx)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if latest == nil {
		t.Fatal("expected S/R record in DB")
	}
	t.Logf("DB record: support=$%.2f resistance=$%.2f midpoint=$%.2f recalc=%v",
		latest.Support, latest.Resistance, latest.Midpoint, latest.GridRecalculated)
}

func TestSRScheduler_WithBotState_PriceOutside(t *testing.T) {
	apiKey := os.Getenv("DUNE_API_KEY")
	if apiKey == "" {
		t.Skip("DUNE_API_KEY not set, skipping")
	}

	pool := testutil.SetupPool(t)
	srRepo := repository.NewSRRepo(pool)
	dune := external.NewDuneClient(apiKey, external.DuneOptions{
		Method:       "simple",
		LookbackDays: 14,
		RefreshHours: 48,
	})

	var recalculated atomic.Bool
	var recalcReasons string

	// Simulate a grid that is far from current market price
	fakeBotState := &scheduler.BotState{
		Grid: []strategy.GridLevel{
			{Index: 0, Price: 1000, Side: "buy"},
			{Index: 1, Price: 1050, Side: "buy"},
			{Index: 2, Price: 1100, Side: "sell"},
			{Index: 3, Price: 1150, Side: "sell"},
		},
		LastETHPrice: 1962, // current real price, way outside the grid
	}

	sched := scheduler.NewSRScheduler(dune, srRepo, scheduler.SRSchedulerConfig{
		CronInterval:      1 * time.Hour,
		SRChangeThreshold: 50, // high threshold so only bot state triggers recalc
		GetBotState: func() *scheduler.BotState {
			return fakeBotState
		},
		OnGridRecalculate: func(sr *external.SRResult) {
			recalculated.Store(true)
			recalcReasons = "price outside grid"
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	err := sched.FetchNow(ctx)
	if err != nil {
		t.Fatalf("FetchNow: %v", err)
	}

	if !recalculated.Load() {
		t.Fatal("expected recalculation due to price outside grid")
	}
	t.Logf("Recalculated: true (reason: %s)", recalcReasons)
}

func TestSRScheduler_StartStop(t *testing.T) {
	pool := testutil.SetupPool(t)
	srRepo := repository.NewSRRepo(pool)

	// Use a dummy Dune client (no API key — won't actually fetch)
	dune := external.NewDuneClient("", external.DuneOptions{})

	sched := scheduler.NewSRScheduler(dune, srRepo, scheduler.SRSchedulerConfig{
		CronInterval: 1 * time.Hour,
	})

	sched.Start()
	if !sched.Running() {
		t.Fatal("expected running after Start")
	}

	// Give initial goroutine a moment (it will fail due to no API key, that's fine)
	time.Sleep(200 * time.Millisecond)

	sched.Stop()
	if sched.Running() {
		t.Fatal("expected not running after Stop")
	}

	t.Log("Start/Stop lifecycle: OK")
}
