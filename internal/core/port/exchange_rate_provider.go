package port

import "context"

// ExchangeRateProvider fetches live exchange rates from an external source.
// Implementations can target different APIs or local files.
type ExchangeRateProvider interface {
	// FetchRates returns rates from base currency to all available currencies.
	// The returned map key is the target currency code (e.g. "EUR").
	FetchRates(ctx context.Context, base string) (map[string]float64, error)
}
