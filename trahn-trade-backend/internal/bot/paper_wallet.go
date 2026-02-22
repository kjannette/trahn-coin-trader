package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/kjannette/trahn-backend/internal/models"
	"github.com/kjannette/trahn-backend/internal/repository"
)

type PaperWallet struct {
	gridRepo    *repository.GridStateRepo
	initialETH  float64
	initialUSDC float64
	ETHBalance  float64
	USDCBalance float64
	Trades      []PaperTrade
	TotalGas    float64
	StartTime   time.Time
}

type PaperTrade struct {
	ID             int       `json:"id"`
	Timestamp      string    `json:"timestamp"`
	Side           string    `json:"side"`
	GridLevel      int       `json:"gridLevel"`
	TriggerPrice   float64   `json:"triggerPrice"`
	ExecutionPrice float64   `json:"executionPrice"`
	ETHAmount      float64   `json:"ethAmount"`
	USDCAmount     float64   `json:"usdcAmount"`
	SlippagePct    float64   `json:"slippagePercent"`
	GasCost        float64   `json:"gasCost"`
	BalanceAfter   BalSnap   `json:"balanceAfter"`
}

type BalSnap struct {
	ETH  float64 `json:"eth"`
	USDC float64 `json:"usdc"`
}

func NewPaperWallet(gridRepo *repository.GridStateRepo, initialETH, initialUSDC float64) *PaperWallet {
	return &PaperWallet{
		gridRepo:    gridRepo,
		initialETH:  initialETH,
		initialUSDC: initialUSDC,
		ETHBalance:  initialETH,
		USDCBalance: initialUSDC,
		StartTime:   time.Now(),
	}
}

func (pw *PaperWallet) Init(ctx context.Context) error {
	state, err := pw.gridRepo.GetPaperWallet(ctx)
	if err != nil {
		return fmt.Errorf("load paper state: %w", err)
	}

	if state != nil && state.ETHBalance > 0 {
		pw.ETHBalance = state.ETHBalance
		pw.USDCBalance = state.USDCBalance
		pw.TotalGas = state.TotalGasSpent
		pw.initialETH = state.InitialETH
		pw.initialUSDC = state.InitialUSDC
		if state.StartTime != nil {
			pw.StartTime = *state.StartTime
		}
		if state.Trades != nil {
			_ = json.Unmarshal(state.Trades, &pw.Trades)
		}
		fmt.Printf("[PAPER] Loaded from DB: %.6f ETH, %.2f USDC, %d trades\n",
			pw.ETHBalance, pw.USDCBalance, len(pw.Trades))
	} else {
		fmt.Printf("[PAPER] Starting fresh paper wallet: %.4f ETH, %.2f USDC\n",
			pw.initialETH, pw.initialUSDC)
		if err := pw.gridRepo.InitializePaperWallet(ctx, pw.initialETH, pw.initialUSDC); err != nil {
			return fmt.Errorf("initialize paper wallet: %w", err)
		}
	}
	return nil
}

func (pw *PaperWallet) save(ctx context.Context) {
	tradesJSON, _ := json.Marshal(pw.Trades)
	err := pw.gridRepo.UpdatePaperWallet(ctx, &models.PaperWallet{
		ETHBalance:    pw.ETHBalance,
		USDCBalance:   pw.USDCBalance,
		TotalGasSpent: pw.TotalGas,
		Trades:        tradesJSON,
	})
	if err != nil {
		fmt.Printf("[PAPER] Failed to save paper state: %v\n", err)
	}
}

func (pw *PaperWallet) ExecuteBuy(ctx context.Context, usdcAmount, ethAmount float64) error {
	if pw.USDCBalance < usdcAmount {
		return fmt.Errorf("insufficient USDC: have %.2f, need %.2f", pw.USDCBalance, usdcAmount)
	}
	pw.USDCBalance -= usdcAmount
	pw.ETHBalance += ethAmount
	pw.save(ctx)
	return nil
}

func (pw *PaperWallet) ExecuteSell(ctx context.Context, ethAmount, usdcAmount float64) error {
	if pw.ETHBalance < ethAmount {
		return fmt.Errorf("insufficient ETH: have %.6f, need %.6f", pw.ETHBalance, ethAmount)
	}
	pw.ETHBalance -= ethAmount
	pw.USDCBalance += usdcAmount
	pw.save(ctx)
	return nil
}

func (pw *PaperWallet) DeductGas(ctx context.Context, gasETH float64) {
	pw.ETHBalance -= gasETH
	pw.TotalGas += gasETH
	pw.save(ctx)
}

func (pw *PaperWallet) RecordTrade(ctx context.Context, t PaperTrade) {
	t.ID = len(pw.Trades) + 1
	t.Timestamp = time.Now().UTC().Format(time.RFC3339)
	t.BalanceAfter = BalSnap{ETH: pw.ETHBalance, USDC: pw.USDCBalance}
	pw.Trades = append(pw.Trades, t)
	pw.save(ctx)
}

type PaperStats struct {
	InitialETH          float64
	InitialUSDC         float64
	CurrentETH          float64
	CurrentUSDC         float64
	InitialValueUSD     float64
	CurrentValueUSD     float64
	UnrealizedPnL       float64
	UnrealizedPnLPct    float64
	TotalTrades         int
	BuyTrades           int
	SellTrades          int
	TotalGasSpent       float64
	GasSpentUSD         float64
	RunningTimeHours    float64
}

func (pw *PaperWallet) Stats(currentETHPrice float64) PaperStats {
	initialVal := pw.initialETH*currentETHPrice + pw.initialUSDC
	currentVal := pw.ETHBalance*currentETHPrice + pw.USDCBalance
	pnl := currentVal - initialVal
	pnlPct := 0.0
	if initialVal > 0 {
		pnlPct = pnl / initialVal * 100
	}
	buys, sells := 0, 0
	for _, t := range pw.Trades {
		if t.Side == "buy" {
			buys++
		} else {
			sells++
		}
	}
	return PaperStats{
		InitialETH:       pw.initialETH,
		InitialUSDC:      pw.initialUSDC,
		CurrentETH:       pw.ETHBalance,
		CurrentUSDC:      pw.USDCBalance,
		InitialValueUSD:  initialVal,
		CurrentValueUSD:  currentVal,
		UnrealizedPnL:    pnl,
		UnrealizedPnLPct: pnlPct,
		TotalTrades:      len(pw.Trades),
		BuyTrades:        buys,
		SellTrades:       sells,
		TotalGasSpent:    pw.TotalGas,
		GasSpentUSD:      pw.TotalGas * currentETHPrice,
		RunningTimeHours: time.Since(pw.StartTime).Hours(),
	}
}

func randomSlippage(maxPct float64) float64 {
	return rand.Float64() * maxPct / 100
}

const defaultPaperGasCost = 0.005
