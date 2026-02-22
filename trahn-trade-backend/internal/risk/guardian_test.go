package risk

import (
	"context"
	"fmt"
	"testing"
)

type mockCounter struct {
	count int
	err   error
}

func (m *mockCounter) CountToday(_ context.Context) (int, error) {
	return m.count, m.err
}

// --- PreTradeCheck ---

func TestPreTradeCheck_PositionSize_Allowed(t *testing.T) {
	g := NewGuardian(Limits{MaxPositionSizeUSD: 500}, &mockCounter{})
	if err := g.PreTradeCheck(context.Background(), 499.99); err != nil {
		t.Fatalf("expected trade to be allowed, got: %v", err)
	}
}

func TestPreTradeCheck_PositionSize_Blocked(t *testing.T) {
	g := NewGuardian(Limits{MaxPositionSizeUSD: 500}, &mockCounter{})
	err := g.PreTradeCheck(context.Background(), 500.01)
	if err == nil {
		t.Fatal("expected trade to be blocked")
	}
	t.Logf("Correctly blocked: %v", err)
}

func TestPreTradeCheck_PositionSize_DisabledWhenZero(t *testing.T) {
	g := NewGuardian(Limits{MaxPositionSizeUSD: 0}, &mockCounter{})
	if err := g.PreTradeCheck(context.Background(), 999999); err != nil {
		t.Fatalf("zero limit should disable check, got: %v", err)
	}
}

func TestPreTradeCheck_DailyTrades_Allowed(t *testing.T) {
	g := NewGuardian(Limits{MaxDailyTrades: 50}, &mockCounter{count: 49})
	if err := g.PreTradeCheck(context.Background(), 100); err != nil {
		t.Fatalf("expected trade to be allowed (49/50), got: %v", err)
	}
}

func TestPreTradeCheck_DailyTrades_Blocked(t *testing.T) {
	g := NewGuardian(Limits{MaxDailyTrades: 50}, &mockCounter{count: 50})
	err := g.PreTradeCheck(context.Background(), 100)
	if err == nil {
		t.Fatal("expected trade to be blocked (50/50)")
	}
	t.Logf("Correctly blocked: %v", err)
}

func TestPreTradeCheck_DailyTrades_CounterError(t *testing.T) {
	g := NewGuardian(Limits{MaxDailyTrades: 50}, &mockCounter{err: fmt.Errorf("db down")})
	err := g.PreTradeCheck(context.Background(), 100)
	if err == nil {
		t.Fatal("expected error when counter fails")
	}
	t.Logf("Correctly blocked on counter error: %v", err)
}

func TestPreTradeCheck_DailyTrades_DisabledWhenZero(t *testing.T) {
	g := NewGuardian(Limits{MaxDailyTrades: 0}, &mockCounter{count: 9999})
	if err := g.PreTradeCheck(context.Background(), 100); err != nil {
		t.Fatalf("zero limit should disable check, got: %v", err)
	}
}

func TestPreTradeCheck_BothChecks_PositionSizeFailsFirst(t *testing.T) {
	g := NewGuardian(Limits{
		MaxPositionSizeUSD: 100,
		MaxDailyTrades:     50,
	}, &mockCounter{count: 49})

	err := g.PreTradeCheck(context.Background(), 200)
	if err == nil {
		t.Fatal("expected trade to be blocked by position size")
	}
	t.Logf("Correctly blocked: %v", err)
}

func TestPreTradeCheck_AllDisabled(t *testing.T) {
	g := NewGuardian(Limits{}, &mockCounter{count: 9999})
	if err := g.PreTradeCheck(context.Background(), 999999); err != nil {
		t.Fatalf("all-zero limits should allow everything, got: %v", err)
	}
}

// --- PortfolioCheck ---

func TestPortfolioCheck_StopLoss_Triggered(t *testing.T) {
	g := NewGuardian(Limits{StopLossPercent: 10}, nil)
	err := g.PortfolioCheck(-10.0)
	if err == nil {
		t.Fatal("expected stop-loss to trigger at -10%")
	}
	t.Logf("Correctly triggered: %v", err)
}

func TestPortfolioCheck_StopLoss_NotTriggered(t *testing.T) {
	g := NewGuardian(Limits{StopLossPercent: 10}, nil)
	if err := g.PortfolioCheck(-9.99); err != nil {
		t.Fatalf("expected no trigger at -9.99%%, got: %v", err)
	}
}

func TestPortfolioCheck_TakeProfit_Triggered(t *testing.T) {
	g := NewGuardian(Limits{TakeProfitPercent: 20}, nil)
	err := g.PortfolioCheck(20.0)
	if err == nil {
		t.Fatal("expected take-profit to trigger at +20%")
	}
	t.Logf("Correctly triggered: %v", err)
}

func TestPortfolioCheck_TakeProfit_NotTriggered(t *testing.T) {
	g := NewGuardian(Limits{TakeProfitPercent: 20}, nil)
	if err := g.PortfolioCheck(19.99); err != nil {
		t.Fatalf("expected no trigger at +19.99%%, got: %v", err)
	}
}

func TestPortfolioCheck_BothDisabled(t *testing.T) {
	g := NewGuardian(Limits{}, nil)
	if err := g.PortfolioCheck(-99); err != nil {
		t.Fatalf("zero limits should disable all checks, got: %v", err)
	}
	if err := g.PortfolioCheck(99); err != nil {
		t.Fatalf("zero limits should disable all checks, got: %v", err)
	}
}

func TestPortfolioCheck_StopLoss_ExactBoundary(t *testing.T) {
	g := NewGuardian(Limits{StopLossPercent: 5}, nil)
	err := g.PortfolioCheck(-5.0)
	if err == nil {
		t.Fatal("expected stop-loss to trigger at exactly -5%")
	}
}

func TestPortfolioCheck_TakeProfit_ExactBoundary(t *testing.T) {
	g := NewGuardian(Limits{TakeProfitPercent: 15}, nil)
	err := g.PortfolioCheck(15.0)
	if err == nil {
		t.Fatal("expected take-profit to trigger at exactly +15%")
	}
}
