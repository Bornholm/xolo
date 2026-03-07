package gorm

import "time"

// ExchangeRate is the GORM model for cached exchange rates.
type ExchangeRate struct {
	FromCurrency string    `gorm:"primaryKey"`
	ToCurrency   string    `gorm:"primaryKey"`
	Rate         float64   `gorm:"not null"`
	FetchedAt    time.Time `gorm:"not null"`
}
