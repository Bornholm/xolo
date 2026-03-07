package model

import "time"

type ExchangeRate struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
	FetchedAt    time.Time
}

func (r ExchangeRate) Convert(amount int64) int64 {
	if r.Rate == 0 {
		return amount
	}
	return int64(float64(amount) * r.Rate)
}
