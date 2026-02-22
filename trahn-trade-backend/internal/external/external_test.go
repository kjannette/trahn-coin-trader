package external_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/kjannette/trahn-backend/internal/external"
)

func init() {
	_ = godotenv.Load("../../.env")
}

func TestCoinGeckoGetETHPrice(t *testing.T) {
	client := external.NewCoinGeckoClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	price, err := client.GetETHPrice(ctx)
	if err != nil {
		t.Fatalf("GetETHPrice: %v", err)
	}
	if price <= 0 {
		t.Fatalf("expected positive price, got %f", price)
	}
	t.Logf("ETH price: $%.2f", price)
}

func TestDuneFetchSupportResistance(t *testing.T) {
	apiKey := os.Getenv("DUNE_API_KEY")
	if apiKey == "" {
		t.Skip("DUNE_API_KEY not set, skipping")
	}

	client := external.NewDuneClient(apiKey, external.DuneOptions{
		Method:       "simple",
		LookbackDays: 14,
		RefreshHours: 48,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	sr, err := client.FetchSupportResistance(ctx, true)
	if err != nil {
		t.Fatalf("FetchSupportResistance: %v", err)
	}

	if sr.Support <= 0 || sr.Resistance <= 0 || sr.Midpoint <= 0 {
		t.Fatalf("invalid S/R values: %+v", sr)
	}
	if sr.Support >= sr.Resistance {
		t.Fatalf("support (%.2f) >= resistance (%.2f)", sr.Support, sr.Resistance)
	}

	t.Logf("Support:    $%.2f", sr.Support)
	t.Logf("Resistance: $%.2f", sr.Resistance)
	t.Logf("Midpoint:   $%.2f", sr.Midpoint)
	t.Logf("AvgPrice:   $%.2f", sr.AvgPrice)
	t.Logf("Method:     %s, Lookback: %d days", sr.Method, sr.LookbackDays)

	// Test cache hit
	sr2, err := client.FetchSupportResistance(ctx, false)
	if err != nil {
		t.Fatalf("cached FetchSupportResistance: %v", err)
	}
	if sr2.Midpoint != sr.Midpoint {
		t.Fatalf("cache mismatch: %.2f != %.2f", sr2.Midpoint, sr.Midpoint)
	}
	t.Log("Cache hit verified")

	// NeedsRefresh should be false right after fetch
	if client.NeedsRefresh() {
		t.Fatal("should not need refresh right after fetch")
	}
	t.Log("NeedsRefresh: false (correct)")
}
