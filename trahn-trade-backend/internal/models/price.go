package models

import "time"

type PricePoint struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Price      float64   `json:"price"`
	TradingDay string    `json:"tradingDay"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"createdAt"`
}
