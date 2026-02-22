package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kjannette/trahn-backend/internal/config"
	"github.com/kjannette/trahn-backend/internal/ethereum"
	"github.com/kjannette/trahn-backend/internal/external"
	"github.com/kjannette/trahn-backend/internal/models"
	"github.com/kjannette/trahn-backend/internal/notifications"
	"github.com/kjannette/trahn-backend/internal/repository"
	"github.com/kjannette/trahn-backend/internal/risk"
	"github.com/kjannette/trahn-backend/internal/strategy"
)

type GridBot struct {
	cfg       *config.Config
	coingecko *external.CoinGeckoClient
	dune      *external.DuneClient
	priceRepo *repository.PriceRepo
	tradeRepo *repository.TradeRepo
	gridRepo  *repository.GridStateRepo
	notify    *notifications.Sender

	Grid             []strategy.GridLevel
	LastETHPrice     float64
	BasePrice        float64
	TradesExecuted   int
	TotalProfit      float64
	PriceChecks      int
	LastStatusReport time.Time
	LastSRRefresh    *time.Time

	guardian     *risk.Guardian
	paperWallet *PaperWallet
	uniswap     *ethereum.UniswapV2
	ethClient   *ethereum.Client

	running bool
	stopCh  chan struct{}
}

func NewGridBot(
	cfg *config.Config,
	priceRepo *repository.PriceRepo,
	tradeRepo *repository.TradeRepo,
	gridRepo *repository.GridStateRepo,
	notify *notifications.Sender,
	dune *external.DuneClient,
) *GridBot {
	b := &GridBot{
		cfg:       cfg,
		coingecko: external.NewCoinGeckoClient(),
		dune:      dune,
		priceRepo: priceRepo,
		tradeRepo: tradeRepo,
		gridRepo:  gridRepo,
		notify:    notify,
		stopCh:    make(chan struct{}),
		guardian: risk.NewGuardian(risk.Limits{
			MaxDailyTrades:     cfg.MaxDailyTrades,
			MaxPositionSizeUSD: cfg.MaxPositionSizeUSD,
			StopLossPercent:    cfg.StopLossPercent,
			TakeProfitPercent:  cfg.TakeProfitPercent,
		}, tradeRepo),
	}

	if dune != nil {
		fmt.Printf("[S/R] Dune Analytics configured: %s method, %d-day lookback\n", cfg.SRMethod, cfg.SRLookbackDays)
	} else {
		fmt.Println("[S/R] Dune API key not set - using fallback (current price as midpoint)")
	}

	return b
}

func (b *GridBot) Init(ctx context.Context) error {
	if err := b.loadState(ctx); err != nil {
		fmt.Printf("Warning: failed to load state: %v\n", err)
	}

	if b.cfg.PaperTradingEnabled {
		b.paperWallet = NewPaperWallet(b.gridRepo, b.cfg.PaperInitialETH, b.cfg.PaperInitialUSDC)
		if err := b.paperWallet.Init(ctx); err != nil {
			return fmt.Errorf("paper wallet init: %w", err)
		}
	} else {
		ethC, err := ethereum.NewClient(
			b.cfg.EthereumAPIEndpoint,
			b.cfg.PrivateKey,
			int64(b.cfg.ChainID),
			b.cfg.GasLimit,
			b.cfg.GasMultiplier,
		)
		if err != nil {
			return fmt.Errorf("ethereum client: %w", err)
		}
		b.ethClient = ethC

		uni, err := ethereum.NewUniswapV2(
			ethC,
			b.cfg.UniswapRouterAddress,
			b.cfg.WETHAddress,
			b.cfg.QuoteTokenAddress,
			b.cfg.QuoteTokenSymbol,
			b.cfg.QuoteTokenDecimals,
			b.cfg.SlippageTolerance,
		)
		if err != nil {
			return fmt.Errorf("uniswap client: %w", err)
		}
		b.uniswap = uni
		fmt.Printf("[LIVE] Ethereum client connected, wallet %s\n", ethC.WalletAddress().Hex())
	}
	return nil
}

// --- state management ---

func (b *GridBot) loadState(ctx context.Context) error {
	state, err := b.gridRepo.GetActive(ctx)
	if err != nil {
		return err
	}
	if state == nil || state.GridLevelsJSON == nil {
		fmt.Println("No existing state found in DB - will initialize fresh")
		return nil
	}

	if err := json.Unmarshal(state.GridLevelsJSON, &b.Grid); err != nil {
		return fmt.Errorf("unmarshal grid: %w", err)
	}
	b.TradesExecuted = state.TradesExecuted
	b.TotalProfit = state.TotalProfit
	if state.BasePrice != nil {
		b.BasePrice = *state.BasePrice
	}
	b.LastSRRefresh = state.LastSRRefresh

	fmt.Printf("Loaded state from DB: %d grid levels, %d trades\n", len(b.Grid), b.TradesExecuted)
	return nil
}

func (b *GridBot) saveState(ctx context.Context) {
	levelsJSON, err := json.Marshal(b.Grid)
	if err != nil {
		fmt.Printf("[STATE] Failed to marshal grid levels: %v\n", err)
		return
	}
	_, err = b.gridRepo.Save(ctx, &models.GridState{
		BasePrice:      &b.BasePrice,
		GridLevelsJSON: levelsJSON,
		TradesExecuted: b.TradesExecuted,
		TotalProfit:    b.TotalProfit,
		LastSRRefresh:  b.LastSRRefresh,
	})
	if err != nil {
		fmt.Printf("[STATE] Failed to save state to DB: %v\n", err)
	}
}

// --- price ---

func (b *GridBot) fetchETHPrice(ctx context.Context) float64 {
	price, err := b.coingecko.GetETHPrice(ctx)
	if err != nil {
		fmt.Printf("Failed to fetch ETH price: %v\n", err)
		return b.LastETHPrice
	}
	if price < 100 || price > 100000 {
		fmt.Printf("ETH price %.2f failed sanity check\n", price)
		return b.LastETHPrice
	}
	b.LastETHPrice = price

	_, _ = b.priceRepo.Record(ctx, price, time.Now())
	return price
}

// --- grid initialization ---

func (b *GridBot) InitializeGrid(ctx context.Context) error {
	sr := b.fetchSR(ctx)

	center := b.BasePrice
	if center == 0 {
		center = sr.Midpoint
	}
	b.BasePrice = center

	price := b.fetchETHPrice(ctx)
	if price <= 0 {
		return fmt.Errorf("cannot initialize grid: invalid ETH price")
	}

	b.notify.Send(fmt.Sprintf("S/R Analysis (%s, %dd): Support $%.2f | Resistance $%.2f | Midpoint $%.2f",
		sr.Method, sr.LookbackDays, sr.Support, sr.Resistance, sr.Midpoint))

	grid, err := strategy.CalculateGridLevels(strategy.GridParams{
		CenterPrice:    center,
		LevelCount:     b.cfg.GridLevels,
		SpacingPercent: b.cfg.GridSpacingPercent,
		AmountPerGrid:  b.cfg.AmountPerGrid,
	})
	if err != nil {
		return fmt.Errorf("calculate grid: %w", err)
	}
	b.Grid = grid
	b.saveState(ctx)

	b.notify.Send(fmt.Sprintf("Grid initialized: %d levels from $%.2f to $%.2f, center at $%.2f",
		len(grid), grid[0].Price, grid[len(grid)-1].Price, center))

	return nil
}

func (b *GridBot) fetchSR(ctx context.Context) *external.SRResult {
	if b.dune == nil {
		price := b.fetchETHPrice(ctx)
		fb := strategy.CreateFallbackSR(price)
		return &external.SRResult{
			Support: fb.Support, Resistance: fb.Resistance,
			Midpoint: fb.Midpoint, Method: fb.Method,
		}
	}

	sr, err := b.dune.FetchSupportResistance(ctx, false)
	if err != nil {
		fmt.Printf("[S/R] Dune fetch failed: %v — falling back to current price\n", err)
		price := b.fetchETHPrice(ctx)
		fb := strategy.CreateFallbackSR(price)
		return &external.SRResult{
			Support: fb.Support, Resistance: fb.Resistance,
			Midpoint: fb.Midpoint, Method: fb.Method,
		}
	}
	now := time.Now()
	b.LastSRRefresh = &now
	return sr
}

// --- trading ---

func (b *GridBot) executeTrade(ctx context.Context, level *strategy.GridLevel, currentPrice float64) error {
	tradeUSD := level.Quantity * currentPrice
	if err := b.guardian.PreTradeCheck(ctx, tradeUSD); err != nil {
		b.notify.Send(fmt.Sprintf("[RISK] %v", err))
		return err
	}

	if level.Side == "buy" {
		return b.executeBuy(ctx, level, currentPrice)
	}
	return b.executeSell(ctx, level, currentPrice)
}

func (b *GridBot) executeBuy(ctx context.Context, level *strategy.GridLevel, currentPrice float64) error {
	return b.executeSwap(ctx, level, currentPrice, "buy")
}

func (b *GridBot) executeSell(ctx context.Context, level *strategy.GridLevel, currentPrice float64) error {
	return b.executeSwap(ctx, level, currentPrice, "sell")
}

func (b *GridBot) executeSwap(ctx context.Context, level *strategy.GridLevel, currentPrice float64, side string) error {
	ethAmount := level.Quantity
	usdcAmount := ethAmount * currentPrice
	prefix := ""
	if b.cfg.PaperTradingEnabled {
		prefix = "[PAPER] "
	}

	b.notify.Send(fmt.Sprintf("%sExecuting %s at grid level %d: ~%.6f ETH for ~%.2f USDC (@ $%.2f/ETH)",
		prefix, side, level.Index, ethAmount, usdcAmount, currentPrice))

	var txHash string
	var slippagePct, gasCost *float64

	if b.cfg.PaperTradingEnabled {
		hash, slip, gas, err := b.executePaperSwap(ctx, level, currentPrice, side, ethAmount, usdcAmount)
		if err != nil {
			return err
		}
		txHash = hash
		slippagePct = slip
		gasCost = gas
	} else {
		hash, gas, err := b.executeLiveSwap(ctx, side, ethAmount, usdcAmount)
		if err != nil {
			return err
		}
		txHash = hash
		gasCost = gas
	}

	level.Filled = true
	now := time.Now()
	level.FilledAt = &now
	level.TxHash = &txHash
	b.TradesExecuted++
	b.saveState(ctx)

	gridLevel := level.Index
	_, _ = b.tradeRepo.Record(ctx, &models.Trade{
		Timestamp:       now,
		Side:            side,
		Price:           currentPrice,
		Quantity:        ethAmount,
		USDValue:        usdcAmount,
		GridLevel:       &gridLevel,
		TxHash:          &txHash,
		IsPaperTrade:    b.cfg.PaperTradingEnabled,
		SlippagePercent: slippagePct,
		GasCostETH:      gasCost,
	})

	b.resetOppositeLevel(ctx, level)
	return nil
}

func (b *GridBot) executePaperSwap(ctx context.Context, level *strategy.GridLevel, currentPrice float64, side string, ethAmount, usdcAmount float64) (txHash string, slippagePct, gasCost *float64, err error) {
	slip := randomSlippage(b.cfg.PaperSlippagePercent)
	gas := defaultPaperGasCost

	if side == "buy" {
		actualETH := ethAmount * (1 - slip)
		if err := b.paperWallet.ExecuteBuy(ctx, usdcAmount, actualETH); err != nil {
			return "", nil, nil, err
		}
		ethAmount = actualETH
	} else {
		if b.paperWallet.ETHBalance < ethAmount+gas {
			return "", nil, nil, fmt.Errorf("insufficient ETH: have %.6f, need %.6f", b.paperWallet.ETHBalance, ethAmount+gas)
		}
		usdcAmount = usdcAmount * (1 - slip)
		if err := b.paperWallet.ExecuteSell(ctx, ethAmount, usdcAmount); err != nil {
			return "", nil, nil, err
		}
	}

	b.paperWallet.DeductGas(ctx, gas)
	b.paperWallet.RecordTrade(ctx, PaperTrade{
		Side: side, GridLevel: level.Index,
		TriggerPrice: level.Price, ExecutionPrice: currentPrice,
		ETHAmount: ethAmount, USDCAmount: usdcAmount,
		SlippagePct: slip * 100, GasCost: gas,
	})

	txHash = fmt.Sprintf("0xPAPER_%s_%x", side, time.Now().UnixNano())
	s := slip * 100
	slippagePct = &s
	gasCost = &gas
	fmt.Printf("[PAPER] %s executed: %.6f ETH for %.2f USDC (slippage: %.3f%%, gas: %.6f ETH)\n",
		side, ethAmount, usdcAmount, s, gas)
	return txHash, slippagePct, gasCost, nil
}

func (b *GridBot) executeLiveSwap(ctx context.Context, side string, ethAmount, usdcAmount float64) (txHash string, gasCost *float64, err error) {
	var hash string
	var swapErr error

	if side == "buy" {
		b.notify.Send(fmt.Sprintf("Broadcasting BUY TX: %.6f ETH for %.2f USDC...", ethAmount, usdcAmount))
		hash, swapErr = b.uniswap.SwapUSDCForETH(ctx, usdcAmount, ethAmount)
	} else {
		b.notify.Send(fmt.Sprintf("Broadcasting SELL TX: %.6f ETH for ~%.2f USDC...", ethAmount, usdcAmount))
		hash, swapErr = b.uniswap.SwapETHForUSDC(ctx, ethAmount)
	}
	if swapErr != nil {
		b.notify.Send(fmt.Sprintf("%s TX failed: %v", side, swapErr))
		return "", nil, fmt.Errorf("swap failed (%s): %w", side, swapErr)
	}

	b.notify.Send(fmt.Sprintf("%s TX confirmed: %s", side, b.uniswap.ExplorerURL(hash)))
	gas, _ := b.uniswap.GasCostETH(ctx)
	gasCost = &gas
	return hash, gasCost, nil
}

func (b *GridBot) resetOppositeLevel(ctx context.Context, filled *strategy.GridLevel) {
	idx := strategy.GetOppositeLevelIndex(filled, len(b.Grid))
	if idx == nil {
		return
	}
	adj := &b.Grid[*idx]
	if adj.Filled {
		adj.Filled = false
		adj.FilledAt = nil
		adj.TxHash = nil
		b.saveState(ctx)
		fmt.Printf("Reset grid level %d for opposite trade\n", *idx)
	}
}

// --- risk ---

// portfolioPnLPercent returns the current unrealized P&L as a percentage.
// The second return value is false when P&L cannot be determined (e.g. live
// mode without balance tracking), in which case the caller should skip the
// portfolio-level check.
func (b *GridBot) portfolioPnLPercent(currentPrice float64) (float64, bool) {
	if b.cfg.PaperTradingEnabled && b.paperWallet != nil {
		return b.paperWallet.Stats(currentPrice).UnrealizedPnLPct, true
	}
	return 0, false
}

// --- main loop ---

func (b *GridBot) Run(ctx context.Context) {
	b.running = true

	b.notify.Send(fmt.Sprintf("Starting ETH grid trader with %d levels, %.1f%% spacing",
		b.cfg.GridLevels, b.cfg.GridSpacingPercent))

	if len(b.Grid) == 0 {
		if err := b.InitializeGrid(ctx); err != nil {
			fmt.Printf("Failed to initialize grid: %v\n", err)
			return
		}
	}

	display := strategy.FormatGridDisplay(b.Grid, b.BasePrice, b.cfg.AmountPerGrid)
	fmt.Println("\n" + display + "\n")

	ticker := time.NewTicker(time.Duration(b.cfg.PriceCheckIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Do one immediate tick
	b.tick(ctx)

	for {
		select {
		case <-b.stopCh:
			b.running = false
			b.notify.Send("Grid trader shutting down")
			return
		case <-ctx.Done():
			b.running = false
			return
		case <-ticker.C:
			b.tick(ctx)
		}
	}
}

func (b *GridBot) tick(ctx context.Context) {
	b.PriceChecks++

	price := b.fetchETHPrice(ctx)
	if price <= 0 {
		fmt.Println("Could not fetch ETH price, skipping tick")
		return
	}

	if pnl, ok := b.portfolioPnLPercent(price); ok {
		if err := b.guardian.PortfolioCheck(pnl); err != nil {
			b.notify.Send(fmt.Sprintf("CIRCUIT BREAKER: %v — halting trading", err))
			fmt.Printf("[RISK] %v\n", err)
			close(b.stopCh)
			b.running = false
			return
		}
	}

	triggered := strategy.FindTriggeredLevel(price, b.Grid)
	if triggered != nil {
		fmt.Printf("Grid level %d triggered: %s ETH at $%.2f\n",
			triggered.Index, triggered.Side, triggered.Price)

		if err := b.executeTrade(ctx, triggered, price); err != nil {
			fmt.Printf("Trade execution failed: %v\n", err)
		} else {
			// Post-trade cooldown
			select {
			case <-time.After(time.Duration(b.cfg.PostTradeCooldownSeconds) * time.Second):
			case <-b.stopCh:
				return
			}
		}
	}

	b.maybeReportStatus(ctx, price)
}

func (b *GridBot) maybeReportStatus(ctx context.Context, currentPrice float64) {
	interval := time.Duration(b.cfg.StatusReportIntervalMinutes) * time.Minute
	if time.Since(b.LastStatusReport) < interval {
		return
	}

	stats := strategy.GetGridStats(b.Grid)
	prefix := ""
	if b.cfg.PaperTradingEnabled {
		prefix = "[PAPER] "
	}

	var ethBal, usdcBal float64
	if b.cfg.PaperTradingEnabled && b.paperWallet != nil {
		ethBal = b.paperWallet.ETHBalance
		usdcBal = b.paperWallet.USDCBalance
	} else if b.uniswap != nil {
		ethBal, _ = b.uniswap.ETHBalance(ctx)
		usdcBal, _ = b.uniswap.TokenBalance(ctx)
	}

	b.notify.Send(fmt.Sprintf(
		"%sStatus: ETH @ $%.2f | ETH: %.4f ($%.2f) | USDC: %.2f | Grid: %d/%d buys, %d/%d sells | Checks: %d | Trades: %d",
		prefix, currentPrice,
		ethBal, ethBal*currentPrice, usdcBal,
		stats.FilledBuys, stats.FilledBuys+stats.PendingBuys,
		stats.FilledSells, stats.FilledSells+stats.PendingSells,
		b.PriceChecks, b.TradesExecuted,
	))

	if b.cfg.PaperTradingEnabled && b.paperWallet != nil {
		ps := b.paperWallet.Stats(currentPrice)
		sign := "+"
		if ps.UnrealizedPnL < 0 {
			sign = ""
		}
		b.notify.Send(fmt.Sprintf(
			"[PAPER P&L] Initial: $%.2f -> Current: $%.2f | P&L: %s$%.2f (%s%.2f%%) | Gas: %.6f ETH ($%.2f) | Running: %.1fh",
			ps.InitialValueUSD, ps.CurrentValueUSD,
			sign, ps.UnrealizedPnL, sign, ps.UnrealizedPnLPct,
			ps.TotalGasSpent, ps.GasSpentUSD, ps.RunningTimeHours,
		))
	}

	b.LastStatusReport = time.Now()
}

func (b *GridBot) Shutdown() {
	if b.running {
		close(b.stopCh)
	}
	if b.ethClient != nil {
		b.ethClient.Close()
	}
	fmt.Println("[BOT] Shutting down gracefully")
}

func (b *GridBot) IsRunning() bool {
	return b.running
}
