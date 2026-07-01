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
	// SumCostSince sums all PAYG (plan_covered=false) costs for a user+org since the given time.
	SumCostSince(ctx context.Context, userID model.UserID, orgID model.OrgID, since time.Time) (int64, error)
	// SumCostSinceByCurrency returns the total PAYG (plan_covered=false) cost per currency for an
	// org (and optionally a subset of users) since the given time. When userIDs is empty, all users
	// are included.
	SumCostSinceByCurrency(ctx context.Context, userIDs []model.UserID, orgID model.OrgID, since time.Time) (map[string]int64, error)
	// SumPlanUsageSince aggregates subscription-covered (plan_covered=true) usage for a specific
	// provider+org since the given time, returning total tokens and total provider-currency value
	// (in microcents of provider currency). Used to enforce rolling-window budgets.
	SumPlanUsageSince(ctx context.Context, orgID model.OrgID, providerID model.ProviderID, since time.Time) (tokens int64, providerValue int64, err error)
	// SumUserPlanUsageSince aggregates subscription-covered (plan_covered=true) usage for a specific
	// user+provider+org since the given time. Used to enforce per-user fair-share rolling-window budgets.
	SumUserPlanUsageSince(ctx context.Context, userID model.UserID, orgID model.OrgID, providerID model.ProviderID, since time.Time) (tokens int64, providerValue int64, err error)
	// EarliestPlanUsageSince returns the creation time of the oldest subscription-covered
	// (plan_covered=true) usage record for a provider+org still inside the rolling window
	// starting at `since`. Used to display when the window will next free up. Returns a zero
	// time when no such record exists.
	EarliestPlanUsageSince(ctx context.Context, orgID model.OrgID, providerID model.ProviderID, since time.Time) (time.Time, error)
}

type UsageFilter struct {
	UserID         *model.UserID
	UserIDs        []model.UserID
	ApplicationID  *model.ApplicationID
	ApplicationIDs []model.ApplicationID
	OrgID          *model.OrgID
	ModelID        *model.LLMModelID
	ProviderID     *model.ProviderID
	AuthTokenID    *string
	Currency       *string
	ProxyModelName *string
	PlanCovered    *bool
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
	CachedTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}
