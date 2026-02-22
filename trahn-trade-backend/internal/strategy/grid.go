package strategy

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type GridLevel struct {
	Index    int        `json:"index"`
	Price    float64    `json:"price"`
	Side     string     `json:"side"` // "buy" or "sell"
	Quantity float64    `json:"quantity"`
	Filled   bool       `json:"filled"`
	FilledAt *time.Time `json:"filledAt,omitempty"`
	TxHash   *string    `json:"txHash,omitempty"`
}

type GridStats struct {
	Levels       int      `json:"levels"`
	LowestPrice  *float64 `json:"lowestPrice"`
	HighestPrice *float64 `json:"highestPrice"`
	FilledLevels int      `json:"filledLevels"`
	PendingBuys  int      `json:"pendingBuys"`
	PendingSells int      `json:"pendingSells"`
	FilledBuys   int      `json:"filledBuys"`
	FilledSells  int      `json:"filledSells"`
}

type FallbackSR struct {
	Support      float64 `json:"support"`
	Resistance   float64 `json:"resistance"`
	Midpoint     float64 `json:"midpoint"`
	Method       string  `json:"method"`
	LookbackDays int     `json:"lookbackDays"`
}

func CalculateMidpoint(support, resistance float64) (float64, error) {
	if support >= resistance {
		return 0, fmt.Errorf("invalid S/R: support (%.2f) >= resistance (%.2f)", support, resistance)
	}
	return (support + resistance) / 2, nil
}

type GridParams struct {
	CenterPrice    float64
	LevelCount     int
	SpacingPercent float64
	AmountPerGrid  float64
}

func CalculateGridLevels(p GridParams) ([]GridLevel, error) {
	if p.CenterPrice <= 0 {
		return nil, fmt.Errorf("center price must be positive")
	}
	if p.LevelCount < 2 {
		return nil, fmt.Errorf("level count must be at least 2")
	}
	if p.SpacingPercent <= 0 {
		return nil, fmt.Errorf("spacing percent must be positive")
	}
	if p.AmountPerGrid <= 0 {
		return nil, fmt.Errorf("amount per grid must be positive")
	}

	halfLevels := p.LevelCount / 2
	even := p.LevelCount%2 == 0

	var grid []GridLevel

	for i := -halfLevels; i <= halfLevels; i++ {
		if i == 0 && even {
			continue
		}

		multiplier := math.Pow(1+p.SpacingPercent/100, float64(i))
		levelPrice := p.CenterPrice * multiplier

		side := "sell"
		if i < 0 {
			side = "buy"
		}

		quantity := p.AmountPerGrid / levelPrice

		grid = append(grid, GridLevel{
			Price:    levelPrice,
			Side:     side,
			Quantity: quantity,
		})
	}

	sort.Slice(grid, func(i, j int) bool {
		return grid[i].Price < grid[j].Price
	})

	for i := range grid {
		grid[i].Index = i
	}

	return grid, nil
}

func FindTriggeredLevel(currentPrice float64, grid []GridLevel) *GridLevel {
	for i := range grid {
		if grid[i].Filled {
			continue
		}
		if grid[i].Side == "buy" && currentPrice <= grid[i].Price {
			return &grid[i]
		}
		if grid[i].Side == "sell" && currentPrice >= grid[i].Price {
			return &grid[i]
		}
	}
	return nil
}

func GetOppositeLevelIndex(filledLevel *GridLevel, gridLength int) *int {
	var idx int
	if filledLevel.Side == "buy" {
		idx = filledLevel.Index + 1
	} else {
		idx = filledLevel.Index - 1
	}
	if idx >= 0 && idx < gridLength {
		return &idx
	}
	return nil
}

func GetGridStats(grid []GridLevel) GridStats {
	if len(grid) == 0 {
		return GridStats{}
	}

	s := GridStats{Levels: len(grid)}
	lo := grid[0].Price
	hi := grid[len(grid)-1].Price
	s.LowestPrice = &lo
	s.HighestPrice = &hi

	for _, l := range grid {
		switch {
		case l.Side == "buy" && l.Filled:
			s.FilledBuys++
			s.FilledLevels++
		case l.Side == "buy" && !l.Filled:
			s.PendingBuys++
		case l.Side == "sell" && l.Filled:
			s.FilledSells++
			s.FilledLevels++
		case l.Side == "sell" && !l.Filled:
			s.PendingSells++
		}
	}
	return s
}

func FormatGridDisplay(grid []GridLevel, centerPrice, amountPerGrid float64) string {
	if len(grid) == 0 {
		return "No grid levels initialized."
	}

	sorted := make([]GridLevel, len(grid))
	copy(sorted, grid)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Price > sorted[j].Price
	})

	var b strings.Builder
	b.WriteString("┌─────────────────────────────────────────────────┐\n")
	b.WriteString("│              GRID LEVELS (USD)               │\n")
	b.WriteString("├─────────────────────────────────────────────────┤\n")

	for _, level := range sorted {
		sideIcon := "BUY "
		if level.Side == "sell" {
			sideIcon = "SELL"
		}
		status := "[ ]"
		if level.Filled {
			status = "[X]"
		}
		fmt.Fprintf(&b, "│ %s %s @ %10.2f │ %15s │\n",
			status, sideIcon, level.Price,
			fmt.Sprintf("%.6f ETH", level.Quantity))
	}

	b.WriteString("├─────────────────────────────────────────────────┤\n")
	fmt.Fprintf(&b, "│  Center: $%8.2f  │  $%.0f/level  │\n", centerPrice, amountPerGrid)
	b.WriteString("└─────────────────────────────────────────────────┘")

	return b.String()
}

func CreateFallbackSR(currentPrice float64) FallbackSR {
	return FallbackSR{
		Support:    currentPrice * 0.9,
		Resistance: currentPrice * 1.1,
		Midpoint:   currentPrice,
		Method:     "fallback",
	}
}

func IsPriceOutsideGrid(currentPrice float64, grid []GridLevel) bool {
	if len(grid) == 0 {
		return true
	}
	lo := grid[0].Price
	hi := grid[0].Price
	for _, l := range grid[1:] {
		if l.Price < lo {
			lo = l.Price
		}
		if l.Price > hi {
			hi = l.Price
		}
	}
	return currentPrice < lo || currentPrice > hi
}

func AreAllSideFilled(grid []GridLevel, side string) bool {
	count := 0
	filled := 0
	for _, l := range grid {
		if l.Side == side {
			count++
			if l.Filled {
				filled++
			}
		}
	}
	if count == 0 {
		return false
	}
	return filled == count
}

func CalculateSRChange(newMidpoint, oldMidpoint float64) float64 {
	if oldMidpoint == 0 {
		return 100
	}
	return math.Abs((newMidpoint - oldMidpoint) / oldMidpoint * 100)
}
