package models

import (
	"math"
	"time"
)

type SupportResistance struct {
	ID               int64     `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Method           string    `json:"method"`
	LookbackDays     int       `json:"lookbackDays"`
	Support          float64   `json:"support"`
	Resistance       float64   `json:"resistance"`
	Midpoint         float64   `json:"midpoint"`
	AvgPrice         *float64  `json:"avgPrice,omitempty"`
	GridRecalculated bool      `json:"gridRecalculated"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (sr *SupportResistance) HasChangedSignificantly(previous *SupportResistance, thresholdPercent float64) bool {
	if previous == nil || previous.Midpoint == 0 {
		return true
	}
	change := math.Abs((sr.Midpoint - previous.Midpoint) / previous.Midpoint * 100)
	return change >= thresholdPercent
}
