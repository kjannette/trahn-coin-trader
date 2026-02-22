package scheduler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/kjannette/trahn-backend/internal/external"
	"github.com/kjannette/trahn-backend/internal/models"
	"github.com/kjannette/trahn-backend/internal/repository"
	"github.com/kjannette/trahn-backend/internal/strategy"
)

// BotState is the subset of bot state the scheduler needs for decision-making.
type BotState struct {
	Grid         []strategy.GridLevel
	LastETHPrice float64
}

// BotStateProvider returns the current bot state, or nil if unavailable.
type BotStateProvider func() *BotState

type SRSchedulerConfig struct {
	CronInterval      time.Duration   // e.g. 1*time.Hour
	SRChangeThreshold float64         // e.g. 5.0 (percent)
	GetBotState       BotStateProvider
	OnSRUpdate        func(sr *external.SRResult)
	OnGridRecalculate func(sr *external.SRResult)
}

type SRScheduler struct {
	dune   *external.DuneClient
	srRepo *repository.SRRepo
	cfg    SRSchedulerConfig

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
}

func NewSRScheduler(dune *external.DuneClient, srRepo *repository.SRRepo, cfg SRSchedulerConfig) *SRScheduler {
	if cfg.CronInterval <= 0 {
		cfg.CronInterval = 1 * time.Hour
	}
	if cfg.SRChangeThreshold <= 0 {
		cfg.SRChangeThreshold = 5
	}
	return &SRScheduler{
		dune:   dune,
		srRepo: srRepo,
		cfg:    cfg,
	}
}

func (s *SRScheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		fmt.Println("[SR-SCHEDULER] Already running")
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	// Initial fetch on startup (fire-and-forget)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := s.fetchAndProcess(ctx); err != nil {
			fmt.Printf("[SR-SCHEDULER] Initial S/R fetch failed: %v\n", err)
		}
	}()

	// Recurring ticker
	go func() {
		ticker := time.NewTicker(s.cfg.CronInterval)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				if err := s.fetchAndProcess(ctx); err != nil {
					fmt.Printf("[SR-SCHEDULER] S/R fetch failed: %v\n", err)
				}
				cancel()
			}
		}
	}()

	fmt.Printf("[SR-SCHEDULER] Started (every %s with intelligent recalculation)\n", s.cfg.CronInterval)
}

func (s *SRScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	close(s.stopCh)
	s.running = false
	fmt.Println("[SR-SCHEDULER] Stopped")
}

func (s *SRScheduler) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// FetchNow manually triggers a fetch outside the normal schedule.
func (s *SRScheduler) FetchNow(ctx context.Context) error {
	fmt.Println("[SR-SCHEDULER] Manual S/R fetch triggered")
	return s.fetchAndProcess(ctx)
}

func (s *SRScheduler) fetchAndProcess(ctx context.Context) error {
	fmt.Println("[SR-SCHEDULER] Fetching S/R levels from Dune...")

	sr, err := s.dune.FetchSupportResistance(ctx, true)
	if err != nil {
		return fmt.Errorf("fetch S/R: %w", err)
	}

	shouldRecalculate := false
	var reasons []string

	// Condition 1: S/R midpoint changed > threshold
	newSR := &models.SupportResistance{Midpoint: sr.Midpoint}
	change, err := s.srRepo.CheckSignificantChange(ctx, newSR, s.cfg.SRChangeThreshold)
	if err != nil {
		fmt.Printf("[SR-SCHEDULER] Warning: could not check S/R change: %v\n", err)
	} else if change.HasChanged {
		shouldRecalculate = true
		pct := ""
		if change.ChangePercent != nil {
			pct = fmt.Sprintf("%.2f%%", *change.ChangePercent)
		}
		reasons = append(reasons, fmt.Sprintf("S/R midpoint changed %s", pct))
	}

	// Conditions 2 & 3: need bot state
	if s.cfg.GetBotState != nil {
		if bot := s.cfg.GetBotState(); bot != nil && len(bot.Grid) > 0 {
			// Condition 2: Price outside grid range
			if bot.LastETHPrice > 0 && strategy.IsPriceOutsideGrid(bot.LastETHPrice, bot.Grid) {
				shouldRecalculate = true
				lo, hi := gridRange(bot.Grid)
				reasons = append(reasons, fmt.Sprintf("Price $%.2f outside grid range ($%.2f - $%.2f)",
					bot.LastETHPrice, lo, hi))
			}

			// Condition 3: All buys or all sells filled
			if strategy.AreAllSideFilled(bot.Grid, "buy") {
				shouldRecalculate = true
				reasons = append(reasons, "All buy levels filled - opportunity to reset")
			}
			if strategy.AreAllSideFilled(bot.Grid, "sell") {
				shouldRecalculate = true
				reasons = append(reasons, "All sell levels filled - opportunity to reset")
			}
		}
	}

	// Store S/R in database
	avgPrice := sr.AvgPrice
	_, err = s.srRepo.Record(ctx, &models.SupportResistance{
		Timestamp:        time.Now(),
		Method:           sr.Method,
		LookbackDays:     sr.LookbackDays,
		Support:          sr.Support,
		Resistance:       sr.Resistance,
		Midpoint:         sr.Midpoint,
		AvgPrice:         &avgPrice,
		GridRecalculated: shouldRecalculate,
	})
	if err != nil {
		return fmt.Errorf("record S/R: %w", err)
	}

	fmt.Printf("[SR-SCHEDULER] S/R stored: Support $%.2f | Resistance $%.2f | Midpoint $%.2f\n",
		sr.Support, sr.Resistance, sr.Midpoint)

	if s.cfg.OnSRUpdate != nil {
		s.cfg.OnSRUpdate(sr)
	}

	if shouldRecalculate {
		fmt.Printf("[SR-SCHEDULER] RECALCULATING GRID - Reasons: %s\n", joinReasons(reasons))
		if s.cfg.OnGridRecalculate != nil {
			s.cfg.OnGridRecalculate(sr)
		}
	} else {
		pctStr := "0"
		if change != nil && change.ChangePercent != nil {
			pctStr = fmt.Sprintf("%.2f", *change.ChangePercent)
		}
		fmt.Printf("[SR-SCHEDULER] Grid stable - no recalculation needed\n")
		fmt.Printf("  S/R change: %s%% (threshold: %.0f%%)\n", pctStr, s.cfg.SRChangeThreshold)
	}

	return nil
}

func gridRange(grid []strategy.GridLevel) (lo, hi float64) {
	lo = math.MaxFloat64
	hi = -math.MaxFloat64
	for _, g := range grid {
		if g.Price < lo {
			lo = g.Price
		}
		if g.Price > hi {
			hi = g.Price
		}
	}
	return
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "none"
	}
	out := reasons[0]
	for _, r := range reasons[1:] {
		out += ", " + r
	}
	return out
}
