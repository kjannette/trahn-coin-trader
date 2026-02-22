package models

import (
	"encoding/json"
	"time"
)

type GridState struct {
	ID             int              `json:"id"`
	BasePrice      *float64         `json:"basePrice,omitempty"`
	GridLevelsJSON json.RawMessage  `json:"gridLevelsJson,omitempty"`
	TradesExecuted int              `json:"tradesExecuted"`
	TotalProfit    float64          `json:"totalProfit"`
	LastSRRefresh  *time.Time       `json:"lastSrRefresh,omitempty"`
	IsActive       bool             `json:"isActive"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`

	// Paper wallet fields (NULL for live trading)
	PaperETHBalance    *float64        `json:"paperEthBalance,omitempty"`
	PaperUSDCBalance   *float64        `json:"paperUsdcBalance,omitempty"`
	PaperTotalGasSpent *float64        `json:"paperTotalGasSpent,omitempty"`
	PaperTradesJSON    json.RawMessage `json:"paperTradesJson,omitempty"`
	PaperStartTime     *time.Time      `json:"paperStartTime,omitempty"`
	PaperInitialETH    *float64        `json:"paperInitialEth,omitempty"`
	PaperInitialUSDC   *float64        `json:"paperInitialUsdc,omitempty"`
}

type PaperWallet struct {
	ETHBalance    float64         `json:"ethBalance"`
	USDCBalance   float64         `json:"usdcBalance"`
	TotalGasSpent float64         `json:"totalGasSpent"`
	Trades        json.RawMessage `json:"trades"`
	StartTime     *time.Time      `json:"startTime"`
	InitialETH    float64         `json:"initialEth"`
	InitialUSDC   float64         `json:"initialUsdc"`
}
