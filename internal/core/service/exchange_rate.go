package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

const DefaultExchangeRateTTL = 24 * time.Hour

// ExchangeRateService resolves exchange rates using a provider (external API or file)
// and a local cache store. It also runs a background refresher goroutine.
type ExchangeRateService struct {
	provider port.ExchangeRateProvider
	store    port.ExchangeRateStore
	ttl      time.Duration
}

func NewExchangeRateService(provider port.ExchangeRateProvider, store port.ExchangeRateStore, ttl time.Duration) *ExchangeRateService {
	return &ExchangeRateService{provider: provider, store: store, ttl: ttl}
}

// Convert converts amount (in microcents) from one currency to another.
// Returns the original amount unchanged if conversion fails, to avoid blocking requests.
func (s *ExchangeRateService) Convert(ctx context.Context, amount int64, from, to string) (int64, error) {
	if from == to || amount == 0 {
		return amount, nil
	}
	rate, err := s.resolveRate(ctx, from, to)
	if err != nil {
		return amount, fmt.Errorf("exchange rate %s→%s unavailable: %w", from, to, err)
	}
	return rate.Convert(amount), nil
}

func (s *ExchangeRateService) resolveRate(ctx context.Context, from, to string) (*model.ExchangeRate, error) {
	cached, err := s.store.GetRate(ctx, from, to)
	if err == nil && cached != nil && time.Since(cached.FetchedAt) < s.ttl {
		return cached, nil
	}
	// Cache miss or stale — fetch fresh rates
	rates, fetchErr := s.provider.FetchRates(ctx, from)
	if fetchErr != nil {
		// Serve stale cache if available rather than failing the request
		if cached != nil {
			slog.Warn("exchange rate fetch failed, using stale cache", slog.String("from", from), slog.String("to", to), slog.Any("error", fetchErr))
			return cached, nil
		}
		return nil, fetchErr
	}
	now := time.Now()
	for toCur, r := range rates {
		_ = s.store.UpsertRate(ctx, model.ExchangeRate{
			FromCurrency: from,
			ToCurrency:   toCur,
			Rate:         r,
			FetchedAt:    now,
		})
	}
	if r, ok := rates[to]; ok {
		return &model.ExchangeRate{FromCurrency: from, ToCurrency: to, Rate: r, FetchedAt: now}, nil
	}
	return nil, fmt.Errorf("no rate found for %s→%s", from, to)
}

// ListRates returns all cached exchange rates from the store.
func (s *ExchangeRateService) ListRates(ctx context.Context) ([]model.ExchangeRate, error) {
	return s.store.ListRates(ctx)
}

// StartRefresher launches a background goroutine that proactively refreshes
// exchange rates for the given base currencies at the given interval.
func (s *ExchangeRateService) StartRefresher(ctx context.Context, baseCurrencies []string, interval time.Duration) {
	go func() {
		s.refreshAll(ctx, baseCurrencies)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.refreshAll(ctx, baseCurrencies)
			}
		}
	}()
}

func (s *ExchangeRateService) refreshAll(ctx context.Context, bases []string) {
	for _, base := range bases {
		rates, err := s.provider.FetchRates(ctx, base)
		if err != nil {
			slog.Warn("exchange rate refresh failed", slog.String("base", base), slog.Any("error", err))
			continue
		}
		now := time.Now()
		for to, r := range rates {
			if err := s.store.UpsertRate(ctx, model.ExchangeRate{
				FromCurrency: base,
				ToCurrency:   to,
				Rate:         r,
				FetchedAt:    now,
			}); err != nil {
				slog.Warn("could not upsert exchange rate", slog.String("from", base), slog.String("to", to), slog.Any("error", err))
			}
		}
	}
}
