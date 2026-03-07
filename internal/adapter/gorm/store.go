package gorm

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Store struct {
	getDatabase func(ctx context.Context) (*gorm.DB, error)
}

func (s *Store) withRetry(ctx context.Context, withTx bool, fn func(ctx context.Context, db *gorm.DB) error, codes ...sqlite3.ErrorCode) error {
	db, err := s.getDatabase(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	backoff := 500 * time.Millisecond
	maxRetries := 10
	retries := 0

	for {
		var err error
		if withTx {
			err = db.Transaction(func(tx *gorm.DB) error {
				if err := fn(ctx, tx); err != nil {
					return errors.WithStack(err)
				}

				return nil
			})
		} else {
			err = fn(ctx, db)
		}

		if err != nil {
			if retries >= maxRetries {
				return errors.WithStack(err)
			}

			var sqliteErr *sqlite3.Error
			if errors.As(err, &sqliteErr) {
				if !slices.Contains(codes, sqliteErr.Code()) {
					return errors.WithStack(err)
				}

				slog.DebugContext(ctx, "transaction failed, will retry", slog.Int("retries", retries), slog.Duration("backoff", backoff), slog.Any("error", errors.WithStack(err)))

				retries++
				time.Sleep(backoff)
				backoff *= 2
				continue
			}

			return errors.WithStack(err)
		}

		return nil
	}
}

func NewStore(db *gorm.DB) *Store {
	return &Store{
		getDatabase: createGetDatabase(db,
			// User store
			&User{}, &AuthToken{}, &UserRole{}, &UserPreferences{},
			// Org store
			&Organization{}, &Membership{},
			// Provider store
			&Provider{}, &LLMModel{},
			// Quota store
			&Quota{},
			// Usage store
			&UsageRecord{},
			// Invite store
			&InviteToken{},
			// Exchange rate cache
			&ExchangeRate{},
		),
	}
}

var _ port.UserStore = &Store{}
