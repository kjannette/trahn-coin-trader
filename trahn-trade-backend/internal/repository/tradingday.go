package repository

import "time"

// TradingDay returns the trading day (YYYY-MM-DD) for a given timestamp.
// Trading day boundary is 12:00 EST (17:00 UTC).
func TradingDay(ts time.Time) string {
	utc := ts.UTC()
	cutoff := 17 * 60 // 17:00 UTC in minutes
	utcMinutes := utc.Hour()*60 + utc.Minute()

	day := utc
	if utcMinutes < cutoff {
		day = day.AddDate(0, 0, -1)
	}
	return day.Format("2006-01-02")
}

// TradingDayNow returns the trading day for the current moment.
func TradingDayNow() string {
	return TradingDay(time.Now())
}
