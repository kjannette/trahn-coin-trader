package models

import "time"

type Trade struct {
	ID              int64     `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	TradingDay      string    `json:"tradingDay"`
	Side            string    `json:"side"` // "buy" or "sell"
	Price           float64   `json:"price"`
	Quantity        float64   `json:"quantity"`
	USDValue        float64   `json:"usdValue"`
	GridLevel       *int      `json:"gridLevel,omitempty"`
	TxHash          *string   `json:"txHash,omitempty"`
	IsPaperTrade    bool      `json:"isPaperTrade"`
	SlippagePercent *float64  `json:"slippagePercent,omitempty"`
	GasCostETH      *float64  `json:"gasCostEth,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type TradeStats struct {
	TotalTrades int64      `json:"totalTrades"`
	BuyCount    int64      `json:"buyCount"`
	SellCount   int64      `json:"sellCount"`
	TotalVolume *float64   `json:"totalVolume"`
	AvgPrice    *float64   `json:"avgPrice"`
	FirstTrade  *time.Time `json:"firstTrade"`
	LastTrade   *time.Time `json:"lastTrade"`
}
