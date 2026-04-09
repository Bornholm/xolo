package gorm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// RecordUsage implements port.UsageStore.
func (s *Store) RecordUsage(ctx context.Context, record model.UsageRecord) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Create(fromUsageRecord(record)).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// QueryUsage implements port.UsageStore.
func (s *Store) QueryUsage(ctx context.Context, filter port.UsageFilter) ([]model.UsageRecord, error) {
	var records []*UsageRecord

	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		query := db.Model(&UsageRecord{})
		query = applyUsageFilter(query, filter)
		query = query.Order("created_at DESC")
		if filter.Limit != nil {
			query = query.Limit(*filter.Limit)
		}
		if filter.Offset != nil {
			query = query.Offset(*filter.Offset)
		}
		return errors.WithStack(query.Find(&records).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}

	result := make([]model.UsageRecord, 0, len(records))
	for _, r := range records {
		result = append(result, &wrappedUsageRecord{r})
	}
	return result, nil
}

// AggregateUsage implements port.UsageStore.
func (s *Store) AggregateUsage(ctx context.Context, filter port.UsageFilter) (*port.UsageAggregate, error) {
	var counts struct {
		TotalRequests    int64
		TotalCost        int64
		PromptTokens     int64
		CompletionTokens int64
		TotalTokens      int64
	}

	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		query := db.Model(&UsageRecord{}).
			Select("COUNT(*) as total_requests, COALESCE(SUM(cost),0) as total_cost, COALESCE(SUM(prompt_tokens),0) as prompt_tokens, COALESCE(SUM(completion_tokens),0) as completion_tokens, COALESCE(SUM(total_tokens),0) as total_tokens")
		query = applyUsageFilter(query, filter)
		return errors.WithStack(query.Scan(&counts).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}

	// Detect the currency from the most recent record matching the filter
	var currency string
	_ = s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		var row struct{ Currency string }
		query := db.Model(&UsageRecord{}).Select("currency").Order("created_at DESC")
		query = applyUsageFilter(query, filter)
		query.Limit(1).Scan(&row)
		currency = row.Currency
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)

	return &port.UsageAggregate{
		TotalRequests:    counts.TotalRequests,
		TotalCost:        counts.TotalCost,
		Currency:         currency,
		PromptTokens:     counts.PromptTokens,
		CompletionTokens: counts.CompletionTokens,
		TotalTokens:      counts.TotalTokens,
	}, nil
}

// SumCostSince implements port.UsageStore.
func (s *Store) SumCostSince(ctx context.Context, userID model.UserID, orgID model.OrgID, since time.Time) (int64, error) {
	var total int64

	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		var result struct{ Total int64 }
		err := db.Model(&UsageRecord{}).
			Select("COALESCE(SUM(cost), 0) as total").
			Where("user_id = ? AND org_id = ? AND created_at >= ?", string(userID), string(orgID), since).
			Scan(&result).Error
		total = result.Total
		return errors.WithStack(err)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func applyUsageFilter(query *gorm.DB, filter port.UsageFilter) *gorm.DB {
	hasUserFilter := filter.UserID != nil || len(filter.UserIDs) > 0
	hasAppFilter := filter.ApplicationID != nil || len(filter.ApplicationIDs) > 0

	if hasUserFilter && hasAppFilter {
		var userConditions []string
		var appConditions []string
		if filter.UserID != nil {
			userConditions = append(userConditions, "user_id = "+string(*filter.UserID))
		}
		if len(filter.UserIDs) > 0 {
			userIDStrings := make([]string, len(filter.UserIDs))
			for i, uid := range filter.UserIDs {
				userIDStrings[i] = fmt.Sprintf("'%s'", uid)
			}
			userConditions = append(userConditions, "user_id IN ("+strings.Join(userIDStrings, ",")+")")
		}
		if filter.ApplicationID != nil {
			appConditions = append(appConditions, "application_id = "+string(*filter.ApplicationID))
		}
		if len(filter.ApplicationIDs) > 0 {
			appIDStrings := make([]string, len(filter.ApplicationIDs))
			for i, aid := range filter.ApplicationIDs {
				appIDStrings[i] = fmt.Sprintf("'%s'", aid)
			}
			appConditions = append(appConditions, "application_id IN ("+strings.Join(appIDStrings, ",")+")")
		}
		userClause := ""
		if len(userConditions) > 0 {
			userClause = "(" + strings.Join(userConditions, " OR ") + ")"
		}
		appClause := ""
		if len(appConditions) > 0 {
			appClause = "(" + strings.Join(appConditions, " OR ") + ")"
		}
		orClause := userClause
		if appClause != "" {
			if orClause != "" {
				orClause += " OR " + appClause
			} else {
				orClause = appClause
			}
		}
		query = query.Where(orClause)
	} else if hasUserFilter {
		if filter.UserID != nil {
			query = query.Where("user_id = ?", string(*filter.UserID))
		}
		if len(filter.UserIDs) > 0 {
			userIDStrings := make([]string, len(filter.UserIDs))
			for i, uid := range filter.UserIDs {
				userIDStrings[i] = string(uid)
			}
			query = query.Where("user_id IN ?", userIDStrings)
		}
	} else if hasAppFilter {
		if filter.ApplicationID != nil {
			query = query.Where("application_id = ?", string(*filter.ApplicationID))
		}
		if len(filter.ApplicationIDs) > 0 {
			appIDStrings := make([]string, len(filter.ApplicationIDs))
			for i, aid := range filter.ApplicationIDs {
				appIDStrings[i] = string(aid)
			}
			query = query.Where("application_id IN ?", appIDStrings)
		}
	}
	if filter.OrgID != nil {
		query = query.Where("org_id = ?", string(*filter.OrgID))
	}
	if filter.ModelID != nil {
		query = query.Where("model_id = ?", string(*filter.ModelID))
	}
	if filter.AuthTokenID != nil {
		query = query.Where("auth_token_id = ?", *filter.AuthTokenID)
	}
	if filter.Currency != nil {
		query = query.Where("currency = ?", *filter.Currency)
	}
	if filter.ProxyModelName != nil {
		query = query.Where("proxy_model_name = ?", *filter.ProxyModelName)
	}
	if filter.Since != nil {
		query = query.Where("created_at >= ?", *filter.Since)
	}
	if filter.Until != nil {
		query = query.Where("created_at <= ?", *filter.Until)
	}
	return query
}

// SumCostSinceByCurrency implements port.UsageStore.
func (s *Store) SumCostSinceByCurrency(ctx context.Context, userIDs []model.UserID, orgID model.OrgID, since time.Time) (map[string]int64, error) {
	var rows []struct {
		Currency string
		Total    int64
	}

	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		query := db.Model(&UsageRecord{}).
			Select("currency, COALESCE(SUM(cost), 0) as total").
			Where("org_id = ? AND created_at >= ?", string(orgID), since)
		if len(userIDs) > 0 {
			ids := make([]string, len(userIDs))
			for i, uid := range userIDs {
				ids[i] = string(uid)
			}
			query = query.Where("user_id IN ?", ids)
		}
		return errors.WithStack(query.Group("currency").Scan(&rows).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Currency] = r.Total
	}
	return result, nil
}

var _ port.UsageStore = &Store{}
