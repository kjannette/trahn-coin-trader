package bot

import (
	"context"
	"fmt"
	"sync"

	"github.com/kjannette/trahn-backend/internal/config"
	"github.com/kjannette/trahn-backend/internal/external"
	"github.com/kjannette/trahn-backend/internal/notifications"
	"github.com/kjannette/trahn-backend/internal/repository"
	"github.com/kjannette/trahn-backend/internal/scheduler"
	"github.com/kjannette/trahn-backend/internal/strategy"
)

type Service struct {
	mu  sync.Mutex
	bot *GridBot
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Start(ctx context.Context, cfg *config.Config,
	priceRepo *repository.PriceRepo,
	tradeRepo *repository.TradeRepo,
	gridRepo *repository.GridStateRepo,
	notify *notifications.Sender,
	dune *external.DuneClient,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bot != nil && s.bot.IsRunning() {
		fmt.Println("[BOT] Already running")
		return nil
	}

	mode := "LIVE MODE"
	if cfg.PaperTradingEnabled {
		mode = "PAPER MODE"
	}
	notify.Send(fmt.Sprintf("Starting ETH Grid Trader (ETH/%s) - %s", cfg.QuoteTokenSymbol, mode))

	b := NewGridBot(cfg, priceRepo, tradeRepo, gridRepo, notify, dune)
	if err := b.Init(ctx); err != nil {
		return fmt.Errorf("bot init: %w", err)
	}
	s.bot = b

	fmt.Println("[BOT] Grid trading bot initialized")
	fmt.Println("[BOT] State loaded from database")

	go func() {
		b.Run(ctx)
		fmt.Println("[BOT] Run loop exited")
	}()

	fmt.Println("[BOT] Started successfully")
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bot != nil {
		s.bot.Shutdown()
		s.bot = nil
	}
	fmt.Println("[BOT] Stopped")
}

// BotState returns the current bot state for the scheduler's decision-making.
func (s *Service) BotState() *scheduler.BotState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bot == nil || len(s.bot.Grid) == 0 {
		return nil
	}

	// Copy grid to avoid data races
	grid := make([]strategy.GridLevel, len(s.bot.Grid))
	copy(grid, s.bot.Grid)

	return &scheduler.BotState{
		Grid:         grid,
		LastETHPrice: s.bot.LastETHPrice,
	}
}

// InitializeGrid triggers a grid recalculation (called by scheduler).
func (s *Service) InitializeGrid(ctx context.Context) {
	s.mu.Lock()
	b := s.bot
	s.mu.Unlock()

	if b == nil {
		return
	}
	fmt.Println("[BOT] Recalculating grid with new S/R midpoint...")
	b.BasePrice = 0 // force recalculation from S/R
	if err := b.InitializeGrid(ctx); err != nil {
		fmt.Printf("[BOT] Grid recalculation failed: %v\n", err)
	}
}
