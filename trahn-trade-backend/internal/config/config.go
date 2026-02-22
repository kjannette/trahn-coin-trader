package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Secrets (from .env)
	DuneAPIKey          string
	WalletAddress       string
	PrivateKey          string
	EthereumAPIEndpoint string
	WebhookURL          string
	BotName             string
	APIKey              string
	CORSAllowOrigin     string

	// Database
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// Blockchain
	ChainID              int
	QuoteTokenAddress    string
	QuoteTokenSymbol     string
	QuoteTokenDecimals   int
	WETHAddress          string
	UniswapRouterAddress string

	// Support/Resistance
	SRMethod       string
	SRRefreshHours int
	SRLookbackDays int

	// Risk Management
	MaxDailyTrades     int
	MaxPositionSizeUSD float64
	StopLossPercent    float64
	TakeProfitPercent  float64

	// Paper Trading
	PaperTradingEnabled  bool
	PaperInitialETH      float64
	PaperInitialUSDC     float64
	PaperSlippagePercent float64
	PaperSimulateGas     bool

	// Grid Configuration
	GridLevels         int
	GridSpacingPercent float64
	GridBasePrice      float64
	AmountPerGrid      float64

	// Trading Parameters
	SlippageTolerance float64
	GasMultiplier     float64
	MinProfitPercent  float64
	GasLimit          int

	// Timing
	PriceCheckIntervalSeconds  int
	StatusReportIntervalMinutes int
	PostTradeCooldownSeconds   int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		// Secrets
		DuneAPIKey:          envStr("DUNE_API_KEY", ""),
		WalletAddress:       envStr("WALLET_ADDRESS", ""),
		PrivateKey:          envStr("PRIVATE_KEY", ""),
		EthereumAPIEndpoint: envStr("ETHEREUM_API_ENDPOINT", ""),
		WebhookURL:          envStr("WEBHOOK_URL", ""),
		BotName:             envStr("BOT_NAME", "TrahnGridTrader"),
		APIKey:              envStr("API_KEY", ""),
		CORSAllowOrigin:     envStr("CORS_ALLOW_ORIGIN", "*"),

		// Database
		DBHost:     envStr("DB_HOST", "localhost"),
		DBPort:     envInt("DB_PORT", 5432),
		DBName:     envStr("DB_NAME", "trahn_grid_trader"),
		DBUser:     envStr("DB_USER", ""),
		DBPassword: envStr("DB_PASSWORD", ""),

		// Blockchain
		ChainID:              envInt("CHAIN_ID", 1),
		QuoteTokenAddress:    envStr("QUOTE_TOKEN_ADDRESS", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		QuoteTokenSymbol:     envStr("QUOTE_TOKEN_SYMBOL", "USDC"),
		QuoteTokenDecimals:   envInt("QUOTE_TOKEN_DECIMALS", 6),
		WETHAddress:          "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
		UniswapRouterAddress: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",

		// Support/Resistance
		SRMethod:       envStr("SR_METHOD", "simple"),
		SRRefreshHours: envInt("SR_REFRESH_HOURS", 48),
		SRLookbackDays: envInt("SR_LOOKBACK_DAYS", 14),

		// Risk Management
		MaxDailyTrades:     envInt("MAX_DAILY_TRADES", 50),
		MaxPositionSizeUSD: envFloat("MAX_POSITION_SIZE_USD", 10000),
		StopLossPercent:    envFloat("STOP_LOSS_PERCENT", 0),
		TakeProfitPercent:  envFloat("TAKE_PROFIT_PERCENT", 0),

		// Paper Trading
		PaperTradingEnabled:  envBool("PAPER_TRADING_ENABLED", true),
		PaperInitialETH:      envFloat("PAPER_INITIAL_ETH", 1.0),
		PaperInitialUSDC:     envFloat("PAPER_INITIAL_USDC", 1000),
		PaperSlippagePercent: envFloat("PAPER_SLIPPAGE_PERCENT", 0.5),
		PaperSimulateGas:     envBool("PAPER_SIMULATE_GAS", true),

		// Grid
		GridLevels:         envInt("GRID_LEVELS", 10),
		GridSpacingPercent: envFloat("GRID_SPACING_PERCENT", 2),
		GridBasePrice:      envFloat("GRID_BASE_PRICE", 0),
		AmountPerGrid:      envFloat("AMOUNT_PER_GRID", 100),

		// Trading Parameters
		SlippageTolerance: envFloat("SLIPPAGE_TOLERANCE", 1.5),
		GasMultiplier:     envFloat("GAS_MULTIPLIER", 1.2),
		MinProfitPercent:  envFloat("MIN_PROFIT_PERCENT", 0.5),
		GasLimit:          envInt("GAS_LIMIT", 250000),

		// Timing
		PriceCheckIntervalSeconds:   envInt("PRICE_CHECK_INTERVAL_SECONDS", 30),
		StatusReportIntervalMinutes: envInt("STATUS_REPORT_INTERVAL_MINUTES", 60),
		PostTradeCooldownSeconds:    envInt("POST_TRADE_COOLDOWN_SECONDS", 60),
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	var errs []string

	if c.WalletAddress == "" {
		errs = append(errs, "WALLET_ADDRESS is required")
	}
	if !c.PaperTradingEnabled && c.PrivateKey == "" {
		errs = append(errs, "PRIVATE_KEY is required for live trading")
	}
	if c.DuneAPIKey == "" {
		fmt.Println("[WARN] DUNE_API_KEY not set — will use current price for grid center (fallback mode)")
	}
	if c.StopLossPercent == 0 && c.TakeProfitPercent == 0 {
		fmt.Println("[WARN] STOP_LOSS_PERCENT and TAKE_PROFIT_PERCENT are both 0 — no portfolio circuit breakers active")
	}
	if c.MaxDailyTrades == 0 && c.MaxPositionSizeUSD == 0 {
		fmt.Println("[WARN] MAX_DAILY_TRADES and MAX_POSITION_SIZE_USD are both 0 — no per-trade limits active")
	}
	if c.APIKey == "" {
		fmt.Println("[WARN] API_KEY not set — REST API has no authentication")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func (c *Config) Print() {
	fmt.Println("=== ETH Grid Trading Bot Configuration ===")

	if c.PaperTradingEnabled {
		fmt.Println("════════════════════════════════════════")
		fmt.Println("  PAPER TRADING MODE ENABLED")
		fmt.Println("  No real transactions will execute")
		fmt.Println("════════════════════════════════════════")
		fmt.Printf("Paper Initial ETH: %.4f\n", c.PaperInitialETH)
		fmt.Printf("Paper Initial %s: %.2f\n", c.QuoteTokenSymbol, c.PaperInitialUSDC)
		fmt.Printf("Paper Slippage: 0-%.1f%%\n", c.PaperSlippagePercent)
		fmt.Printf("Paper Gas Simulation: %v\n", c.PaperSimulateGas)
	} else {
		fmt.Println("  LIVE TRADING MODE")
	}

	fmt.Println("--------------------------------------")
	fmt.Printf("Chain ID: %d\n", c.ChainID)
	if len(c.WalletAddress) > 16 {
		fmt.Printf("Wallet: %s...%s\n", c.WalletAddress[:10], c.WalletAddress[len(c.WalletAddress)-6:])
	}
	fmt.Printf("Trading Pair: ETH/%s\n", c.QuoteTokenSymbol)
	fmt.Printf("Quote Token: %s (%s...)\n", c.QuoteTokenSymbol, truncAddr(c.QuoteTokenAddress))
	fmt.Println("--------------------------------------")
	fmt.Println("Grid Configuration:")
	fmt.Printf("  Levels: %d\n", c.GridLevels)
	fmt.Printf("  Spacing: %.1f%%\n", c.GridSpacingPercent)
	fmt.Printf("  Amount/Grid: $%.0f\n", c.AmountPerGrid)
	fmt.Println("--------------------------------------")
	fmt.Println("Support/Resistance Configuration:")
	fmt.Printf("  S/R Method: %s\n", c.SRMethod)
	fmt.Printf("  S/R Refresh: every %d hours\n", c.SRRefreshHours)
	fmt.Printf("  S/R Lookback: %d days\n", c.SRLookbackDays)
	fmt.Printf("  Dune API: %s\n", boolLabel(c.DuneAPIKey != "", "configured", "not set (fallback mode)"))
	fmt.Println("======================================")
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// --- helpers ---

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		v = strings.ToLower(v)
		return v == "true" || v == "1" || v == "yes"
	}
	return fallback
}

func truncAddr(addr string) string {
	if len(addr) > 10 {
		return addr[:10]
	}
	return addr
}

func boolLabel(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}
