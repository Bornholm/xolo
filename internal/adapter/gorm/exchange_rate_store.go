package gorm

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExchangeRateStore struct {
	db *gorm.DB
}

func NewExchangeRateStore(db *gorm.DB) *ExchangeRateStore {
	return &ExchangeRateStore{db: db}
}

func (s *ExchangeRateStore) GetRate(ctx context.Context, from, to string) (*model.ExchangeRate, error) {
	var row ExchangeRate
	if err := s.db.WithContext(ctx).First(&row, "from_currency = ? AND to_currency = ?", from, to).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrNotFound
		}
		return nil, errors.WithStack(err)
	}
	r := model.ExchangeRate{
		FromCurrency: row.FromCurrency,
		ToCurrency:   row.ToCurrency,
		Rate:         row.Rate,
		FetchedAt:    row.FetchedAt,
	}
	return &r, nil
}

func (s *ExchangeRateStore) ListRates(ctx context.Context) ([]model.ExchangeRate, error) {
	var rows []ExchangeRate
	if err := s.db.WithContext(ctx).Order("from_currency, to_currency").Find(&rows).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	result := make([]model.ExchangeRate, len(rows))
	for i, r := range rows {
		result[i] = model.ExchangeRate{
			FromCurrency: r.FromCurrency,
			ToCurrency:   r.ToCurrency,
			Rate:         r.Rate,
			FetchedAt:    r.FetchedAt,
		}
	}
	return result, nil
}

func (s *ExchangeRateStore) UpsertRate(ctx context.Context, rate model.ExchangeRate) error {
	row := ExchangeRate{
		FromCurrency: rate.FromCurrency,
		ToCurrency:   rate.ToCurrency,
		Rate:         rate.Rate,
		FetchedAt:    rate.FetchedAt,
	}
	return errors.WithStack(s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "from_currency"}, {Name: "to_currency"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate", "fetched_at"}),
	}).Create(&row).Error)
}

var _ port.ExchangeRateStore = &ExchangeRateStore{}
