package port

import (
	"context"

	"github.com/xolo-gateway/xolo/internal/core/model"
)

type ExchangeRateStore interface {
	GetRate(ctx context.Context, from, to string) (*model.ExchangeRate, error)
	UpsertRate(ctx context.Context, rate model.ExchangeRate) error
	ListRates(ctx context.Context) ([]model.ExchangeRate, error)
}
