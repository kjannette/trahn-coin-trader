package strategy

import (
	"math"
	"testing"
)

func TestCalculateMidpoint(t *testing.T) {
	mid, err := CalculateMidpoint(2400, 3000)
	if err != nil {
		t.Fatal(err)
	}
	if mid != 2700 {
		t.Fatalf("expected 2700, got %f", mid)
	}

	_, err = CalculateMidpoint(3000, 2400)
	if err == nil {
		t.Fatal("expected error for support >= resistance")
	}

	_, err = CalculateMidpoint(2500, 2500)
	if err == nil {
		t.Fatal("expected error for support == resistance")
	}
}

func TestCalculateGridLevels(t *testing.T) {
	grid, err := CalculateGridLevels(GridParams{
		CenterPrice:    2700,
		LevelCount:     10,
		SpacingPercent: 2,
		AmountPerGrid:  100,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(grid) != 10 {
		t.Fatalf("expected 10 levels, got %d", len(grid))
	}

	// Sorted ascending
	for i := 1; i < len(grid); i++ {
		if grid[i].Price <= grid[i-1].Price {
			t.Fatalf("not sorted ascending at index %d: %.2f <= %.2f", i, grid[i].Price, grid[i-1].Price)
		}
	}

	// Indices sequential
	for i, l := range grid {
		if l.Index != i {
			t.Fatalf("index mismatch at %d: got %d", i, l.Index)
		}
	}

	// Lower half = buy, upper half = sell
	buys := 0
	sells := 0
	for _, l := range grid {
		if l.Side == "buy" {
			buys++
		} else {
			sells++
		}
		if l.Quantity <= 0 {
			t.Fatalf("quantity must be positive: %.6f", l.Quantity)
		}
		if l.Filled {
			t.Fatal("new levels should not be filled")
		}
	}
	if buys != 5 || sells != 5 {
		t.Fatalf("expected 5 buys + 5 sells, got %d buys + %d sells", buys, sells)
	}

	// Buy levels should have prices below center, sell above
	for _, l := range grid {
		if l.Side == "buy" && l.Price >= 2700 {
			t.Fatalf("buy level at %.2f should be below center 2700", l.Price)
		}
		if l.Side == "sell" && l.Price <= 2700 {
			t.Fatalf("sell level at %.2f should be above center 2700", l.Price)
		}
	}

	t.Logf("Grid levels (center=2700, 10 levels, 2%% spacing):")
	for _, l := range grid {
		t.Logf("  [%d] %s @ $%.2f  qty=%.6f ETH", l.Index, l.Side, l.Price, l.Quantity)
	}
}

func TestCalculateGridLevels_OddCount(t *testing.T) {
	grid, err := CalculateGridLevels(GridParams{
		CenterPrice:    2000,
		LevelCount:     7,
		SpacingPercent: 3,
		AmountPerGrid:  50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(grid) != 7 {
		t.Fatalf("expected 7 levels, got %d", len(grid))
	}
	t.Logf("Odd grid: %d levels", len(grid))
}

func TestCalculateGridLevels_Validation(t *testing.T) {
	cases := []GridParams{
		{CenterPrice: -1, LevelCount: 10, SpacingPercent: 2, AmountPerGrid: 100},
		{CenterPrice: 2700, LevelCount: 1, SpacingPercent: 2, AmountPerGrid: 100},
		{CenterPrice: 2700, LevelCount: 10, SpacingPercent: 0, AmountPerGrid: 100},
		{CenterPrice: 2700, LevelCount: 10, SpacingPercent: 2, AmountPerGrid: -5},
	}
	for i, c := range cases {
		_, err := CalculateGridLevels(c)
		if err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestFindTriggeredLevel(t *testing.T) {
	grid := []GridLevel{
		{Index: 0, Price: 2550, Side: "buy"},
		{Index: 1, Price: 2600, Side: "buy"},
		{Index: 2, Price: 2700, Side: "sell"},
		{Index: 3, Price: 2750, Side: "sell"},
	}

	// Price at 2540 triggers buy at 2550 (index 0)
	triggered := FindTriggeredLevel(2540, grid)
	if triggered == nil {
		t.Fatal("expected a triggered level")
	}
	if triggered.Index != 0 {
		t.Fatalf("expected index 0 (buy at 2550), got %d", triggered.Index)
	}

	// Price at 2590 triggers buy at 2600 (index 1), not 2550
	triggered = FindTriggeredLevel(2590, grid)
	if triggered == nil {
		t.Fatal("expected a triggered level")
	}
	if triggered.Index != 1 {
		t.Fatalf("expected index 1 (buy at 2600), got %d", triggered.Index)
	}

	// Price at 2710 triggers sell at 2700
	triggered = FindTriggeredLevel(2710, grid)
	if triggered == nil {
		t.Fatal("expected a triggered level")
	}
	if triggered.Index != 2 {
		t.Fatalf("expected index 2 (sell at 2700), got %d", triggered.Index)
	}

	// Price at 2650 — no trigger (between buy and sell)
	triggered = FindTriggeredLevel(2650, grid)
	if triggered != nil {
		t.Fatalf("expected no trigger at 2650, got index %d", triggered.Index)
	}

	// Filled levels are skipped
	grid[0].Filled = true
	triggered = FindTriggeredLevel(2540, grid)
	if triggered == nil {
		t.Fatal("expected triggered level")
	}
	if triggered.Index != 1 {
		t.Fatalf("expected index 1 (skipping filled 0), got %d", triggered.Index)
	}
}

func TestGetOppositeLevelIndex(t *testing.T) {
	buy := &GridLevel{Index: 2, Side: "buy"}
	sell := &GridLevel{Index: 3, Side: "sell"}

	idx := GetOppositeLevelIndex(buy, 6)
	if idx == nil || *idx != 3 {
		t.Fatalf("buy at 2: expected opposite 3, got %v", idx)
	}

	idx = GetOppositeLevelIndex(sell, 6)
	if idx == nil || *idx != 2 {
		t.Fatalf("sell at 3: expected opposite 2, got %v", idx)
	}

	// Out of bounds
	edge := &GridLevel{Index: 0, Side: "sell"}
	idx = GetOppositeLevelIndex(edge, 5)
	if idx != nil {
		t.Fatalf("expected nil for out-of-bounds, got %d", *idx)
	}

	top := &GridLevel{Index: 4, Side: "buy"}
	idx = GetOppositeLevelIndex(top, 5)
	if idx != nil {
		t.Fatalf("expected nil for out-of-bounds, got %d", *idx)
	}
}

func TestGetGridStats(t *testing.T) {
	grid := []GridLevel{
		{Index: 0, Price: 2500, Side: "buy", Filled: true},
		{Index: 1, Price: 2600, Side: "buy", Filled: false},
		{Index: 2, Price: 2700, Side: "sell", Filled: false},
		{Index: 3, Price: 2800, Side: "sell", Filled: true},
	}

	s := GetGridStats(grid)
	if s.Levels != 4 {
		t.Fatalf("expected 4 levels, got %d", s.Levels)
	}
	if s.FilledBuys != 1 || s.PendingBuys != 1 {
		t.Fatalf("buys: filled=%d pending=%d", s.FilledBuys, s.PendingBuys)
	}
	if s.FilledSells != 1 || s.PendingSells != 1 {
		t.Fatalf("sells: filled=%d pending=%d", s.FilledSells, s.PendingSells)
	}
	if *s.LowestPrice != 2500 || *s.HighestPrice != 2800 {
		t.Fatalf("price range: %.2f - %.2f", *s.LowestPrice, *s.HighestPrice)
	}

	// Empty grid
	empty := GetGridStats(nil)
	if empty.Levels != 0 {
		t.Fatal("expected 0 levels for nil grid")
	}
}

func TestFormatGridDisplay(t *testing.T) {
	grid := []GridLevel{
		{Index: 0, Price: 2600, Side: "buy", Quantity: 0.0385},
		{Index: 1, Price: 2700, Side: "sell", Quantity: 0.0370, Filled: true},
	}
	out := FormatGridDisplay(grid, 2650, 100)
	if out == "" {
		t.Fatal("expected non-empty display")
	}
	t.Logf("\n%s", out)

	empty := FormatGridDisplay(nil, 0, 0)
	if empty != "No grid levels initialized." {
		t.Fatalf("expected empty message, got: %s", empty)
	}
}

func TestCreateFallbackSR(t *testing.T) {
	sr := CreateFallbackSR(2000)
	if sr.Support != 1800 {
		t.Fatalf("expected support 1800, got %.2f", sr.Support)
	}
	if sr.Resistance != 2200 {
		t.Fatalf("expected resistance 2200, got %.2f", sr.Resistance)
	}
	if sr.Midpoint != 2000 {
		t.Fatalf("expected midpoint 2000, got %.2f", sr.Midpoint)
	}
	if sr.Method != "fallback" {
		t.Fatalf("expected method fallback, got %s", sr.Method)
	}
}

func TestIsPriceOutsideGrid(t *testing.T) {
	grid := []GridLevel{
		{Price: 2500},
		{Price: 2600},
		{Price: 2700},
		{Price: 2800},
	}

	if IsPriceOutsideGrid(2650, grid) {
		t.Fatal("2650 should be inside [2500, 2800]")
	}
	if !IsPriceOutsideGrid(2400, grid) {
		t.Fatal("2400 should be outside")
	}
	if !IsPriceOutsideGrid(2900, grid) {
		t.Fatal("2900 should be outside")
	}
	if !IsPriceOutsideGrid(1000, nil) {
		t.Fatal("empty grid should return outside")
	}
}

func TestAreAllSideFilled(t *testing.T) {
	grid := []GridLevel{
		{Side: "buy", Filled: true},
		{Side: "buy", Filled: true},
		{Side: "sell", Filled: false},
		{Side: "sell", Filled: true},
	}

	if !AreAllSideFilled(grid, "buy") {
		t.Fatal("all buys are filled")
	}
	if AreAllSideFilled(grid, "sell") {
		t.Fatal("not all sells are filled")
	}
	if AreAllSideFilled(nil, "buy") {
		t.Fatal("empty grid should return false")
	}
}

func TestCalculateSRChange(t *testing.T) {
	pct := CalculateSRChange(2800, 2700)
	expected := math.Abs((2800 - 2700) / 2700.0 * 100)
	if math.Abs(pct-expected) > 0.001 {
		t.Fatalf("expected %.4f, got %.4f", expected, pct)
	}

	pct = CalculateSRChange(2700, 0)
	if pct != 100 {
		t.Fatalf("expected 100 for zero old midpoint, got %.2f", pct)
	}

	pct = CalculateSRChange(2700, 2700)
	if pct != 0 {
		t.Fatalf("expected 0 for no change, got %.2f", pct)
	}
}
