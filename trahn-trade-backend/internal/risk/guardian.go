package risk

import (
	"context"
	"fmt"
)

// DailyTradeCounter abstracts the trade-counting dependency so Guardian
// can be tested without a real database.
type DailyTradeCounter interface {
	CountToday(ctx context.Context) (int, error)
}

// Limits holds the four risk thresholds from config.
// A zero value for any field means that check is disabled.
type Limits struct {
	MaxDailyTrades     int
	MaxPositionSizeUSD float64
	StopLossPercent    float64
	TakeProfitPercent  float64
}

type Guardian struct {
	limits  Limits
	counter DailyTradeCounter
}

func NewGuardian(limits Limits, counter DailyTradeCounter) *Guardian {
	return &Guardian{limits: limits, counter: counter}
}

// PreTradeCheck validates per-trade constraints before execution.
// Returns nil if the trade is allowed, a descriptive error if blocked.
func (g *Guardian) PreTradeCheck(ctx context.Context, tradeUSDValue float64) error {
	if g.limits.MaxPositionSizeUSD > 0 && tradeUSDValue > g.limits.MaxPositionSizeUSD {
		return fmt.Errorf("trade blocked: position size $%.2f exceeds max $%.2f",
			tradeUSDValue, g.limits.MaxPositionSizeUSD)
	}

	if g.limits.MaxDailyTrades > 0 && g.counter != nil {
		count, err := g.counter.CountToday(ctx)
		if err != nil {
			return fmt.Errorf("trade blocked: unable to verify daily trade count: %w", err)
		}
		if count >= g.limits.MaxDailyTrades {
			return fmt.Errorf("trade blocked: daily limit of %d trades reached (%d executed today)",
				g.limits.MaxDailyTrades, count)
		}
	}

	return nil
}

// PortfolioCheck evaluates portfolio-level circuit breakers.
// pnlPercent is the unrealized P&L as a percentage (e.g. -8.5 means down 8.5%).
// Returns nil if trading should continue, a descriptive error if a breaker tripped.
func (g *Guardian) PortfolioCheck(pnlPercent float64) error {
	if g.limits.StopLossPercent > 0 && pnlPercent <= -g.limits.StopLossPercent {
		return fmt.Errorf("STOP-LOSS triggered: portfolio down %.2f%% (threshold: -%.2f%%)",
			pnlPercent, g.limits.StopLossPercent)
	}

	if g.limits.TakeProfitPercent > 0 && pnlPercent >= g.limits.TakeProfitPercent {
		return fmt.Errorf("TAKE-PROFIT triggered: portfolio up %.2f%% (threshold: +%.2f%%)",
			pnlPercent, g.limits.TakeProfitPercent)
	}

	return nil
}
