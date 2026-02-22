package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kjannette/trahn-backend/internal/api"
	"github.com/kjannette/trahn-backend/internal/bot"
	"github.com/kjannette/trahn-backend/internal/config"
	"github.com/kjannette/trahn-backend/internal/db"
	"github.com/kjannette/trahn-backend/internal/external"
	"github.com/kjannette/trahn-backend/internal/notifications"
	"github.com/kjannette/trahn-backend/internal/repository"
	"github.com/kjannette/trahn-backend/internal/scheduler"
)

const banner = `
╔══════════════════════════════════════╗
║     TRAHN Grid Trading Bot v0.2      ║
║                                      ║
╚══════════════════════════════════════╝
`

const apiPort = 3001

func main() {
	fmt.Print(banner)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	cfg.Print()

	// Database
	fmt.Printf("\n[DB] Connecting to %s:%d/%s ...\n", cfg.DBHost, cfg.DBPort, cfg.DBName)
	pool, err := db.Connect(cfg.DSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DB] Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		pool.Close()
		fmt.Println("[DB] Connection pool closed")
	}()

	if err := db.TestConnection(pool); err != nil {
		fmt.Fprintf(os.Stderr, "[DB] Test query failed: %v\n", err)
		os.Exit(1)
	}

	// Repos
	priceRepo := repository.NewPriceRepo(pool)
	tradeRepo := repository.NewTradeRepo(pool)
	srRepo := repository.NewSRRepo(pool)
	gridRepo := repository.NewGridStateRepo(pool)

	// Shared Dune client (single instance for bot + scheduler)
	var dune *external.DuneClient
	if cfg.DuneAPIKey != "" {
		dune = external.NewDuneClient(cfg.DuneAPIKey, external.DuneOptions{
			Method:       cfg.SRMethod,
			LookbackDays: cfg.SRLookbackDays,
			RefreshHours: cfg.SRRefreshHours,
		})

		// Warm cache from DB if a recent S/R record exists
		if latest, err := srRepo.GetLatest(context.Background()); err == nil && latest != nil {
			dune.SeedCache(&external.SRResult{
				Support:      latest.Support,
				Resistance:   latest.Resistance,
				Midpoint:     latest.Midpoint,
				Method:       latest.Method,
				LookbackDays: latest.LookbackDays,
				FetchedAt:    latest.Timestamp,
			})
		}
	}

	// Notifications
	notify := notifications.NewSender(cfg.WebhookURL, cfg.BotName)

	// Graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. API server
	srv := api.NewServer(pool, apiPort, cfg.APIKey, cfg.CORSAllowOrigin)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "[API] Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// 2. Grid bot (shares the Dune client)
	botService := bot.NewService()
	if err := botService.Start(ctx, cfg, priceRepo, tradeRepo, gridRepo, notify, dune); err != nil {
		fmt.Fprintf(os.Stderr, "[BOT] Start failed: %v\n", err)
		os.Exit(1)
	}

	// 3. S/R Scheduler (shares the same Dune client)
	var srSched *scheduler.SRScheduler
	if dune != nil {
		srSched = scheduler.NewSRScheduler(dune, srRepo, scheduler.SRSchedulerConfig{
			CronInterval:      1 * time.Hour,
			SRChangeThreshold: 5,
			GetBotState:       botService.BotState,
			OnGridRecalculate: func(sr *external.SRResult) {
				botService.InitializeGrid(ctx)
			},
		})
		srSched.Start()
	} else {
		fmt.Println("[SCHEDULER] Skipped - no Dune API key configured")
	}

	fmt.Println("\nAll services started successfully")

	// Wait for shutdown signal
	<-ctx.Done()
	fmt.Println("\nShutting down gracefully...")

	if srSched != nil {
		srSched.Stop()
	}

	botService.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "[API] Shutdown error: %v\n", err)
	}
	fmt.Println("[API] Server closed")
	fmt.Println("Shutdown complete")
}
