package port

import (
	"context"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type UsageStore interface {
	RecordUsage(ctx context.Context, record model.UsageRecord) error
	QueryUsage(ctx context.Context, filter UsageFilter) ([]model.UsageRecord, error)
	AggregateUsage(ctx context.Context, filter UsageFilter) (*UsageAggregate, error)
	// SumCostSince sums all costs for a user+org since the given time,
	// regardless of the currency in which individual records were stored.
	SumCostSince(ctx context.Context, userID model.UserID, orgID model.OrgID, since time.Time) (int64, error)
	// SumCostSinceByCurrency returns the total cost per currency for an org
	// (and optionally a subset of users) since the given time, so callers can
	// convert each currency independently. When userIDs is empty, all users are included.
	SumCostSinceByCurrency(ctx context.Context, userIDs []model.UserID, orgID model.OrgID, since time.Time) (map[string]int64, error)
}

type UsageFilter struct {
	UserID         *model.UserID
	UserIDs        []model.UserID // pour filtrer par plusieurs utilisateurs
	OrgID          *model.OrgID
	ModelID        *model.LLMModelID
	AuthTokenID    *string
	Currency       *string
	ProxyModelName *string
	Since          *time.Time
	Until          *time.Time
	Limit          *int
	Offset         *int
}

type UsageAggregate struct {
	TotalRequests    int64
	TotalCost        int64  // microcents, in org's currency
	Currency         string // org's currency
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}
