package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// planScope identifies the org+provider pair for a subscription plan constraint.
type planScope struct {
	OrgID      model.OrgID
	ProviderID model.ProviderID
}

// planDenial describes why a constraint blocked a request.
type planDenial struct {
	Message    string
	RetryAfter time.Duration
}

// planReservation is returned by a successful Acquire; must be Released after the request.
type planReservation interface {
	Release(ctx context.Context)
}

type noopReservation struct{}

func (noopReservation) Release(_ context.Context) {}

// concurrencyReservation decrements the in-flight counter on release.
type concurrencyReservation struct {
	state      *SubscriptionStateDetailed
	orgID      model.OrgID
	providerID model.ProviderID
}

func (r *concurrencyReservation) Release(_ context.Context) {
	r.state.DecrementInFlight(r.orgID, r.providerID)
}

// constraintEvaluator evaluates a single plan constraint and either grants or denies access.
type constraintEvaluator interface {
	Kind() model.PlanConstraintKind
	// Acquire checks the constraint; on success it returns a reservation (may be no-op).
	// On failure it returns a denial with the reason.
	Acquire(ctx context.Context, scope planScope, c model.PlanConstraint) (planReservation, *planDenial, error)
}

// rollingWindowEvaluator enforces time-based rolling budgets (token count and/or value).
type rollingWindowEvaluator struct {
	usageStore port.UsageStore
}

func (e *rollingWindowEvaluator) Kind() model.PlanConstraintKind { return model.ConstraintRollingWindow }

func (e *rollingWindowEvaluator) Acquire(ctx context.Context, scope planScope, c model.PlanConstraint) (planReservation, *planDenial, error) {
	dur := c.Duration.Duration()
	if dur <= 0 || (c.TokenBudget == nil && c.ValueBudget == nil) {
		return noopReservation{}, nil, nil
	}

	since := time.Now().Add(-dur)
	tokens, providerValue, err := e.usageStore.SumPlanUsageSince(ctx, scope.OrgID, scope.ProviderID, since)
	if err != nil {
		return nil, nil, err
	}

	if c.TokenBudget != nil && tokens >= *c.TokenBudget {
		return nil, &planDenial{
			Message: fmt.Sprintf("plan quota exceeded [%s]: %d / %d tokens used in the last %s",
				c.Label, tokens, *c.TokenBudget, formatDuration(dur)),
		}, nil
	}

	if c.ValueBudget != nil && providerValue >= *c.ValueBudget {
		return nil, &planDenial{
			Message: fmt.Sprintf("plan quota exceeded [%s]: value budget of %s reached in the last %s",
				c.Label, formatMicrocents(providerValue, "USD"), formatDuration(dur)),
		}, nil
	}

	return noopReservation{}, nil, nil
}

// concurrencyEvaluator enforces a maximum number of simultaneous in-flight requests.
type concurrencyEvaluator struct {
	state *SubscriptionStateDetailed
}

func (e *concurrencyEvaluator) Kind() model.PlanConstraintKind { return model.ConstraintConcurrency }

func (e *concurrencyEvaluator) Acquire(_ context.Context, scope planScope, c model.PlanConstraint) (planReservation, *planDenial, error) {
	if c.MaxConcurrent == nil || *c.MaxConcurrent <= 0 {
		return noopReservation{}, nil, nil
	}

	// Increment first, then check — this prevents thundering herd but may momentarily
	// exceed the limit by 1. We decrement immediately if the limit is breached.
	current := e.state.IncrementInFlight(scope.OrgID, scope.ProviderID)
	if current > *c.MaxConcurrent {
		e.state.DecrementInFlight(scope.OrgID, scope.ProviderID)
		return nil, &planDenial{
			Message: fmt.Sprintf("concurrency limit reached [%s]: %d / %d concurrent requests",
				c.Label, current-1, *c.MaxConcurrent),
		}, nil
	}

	return &concurrencyReservation{
		state:      e.state,
		orgID:      scope.OrgID,
		providerID: scope.ProviderID,
	}, nil, nil
}

func formatDuration(d time.Duration) string {
	switch {
	case d%time.Hour == 0:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d%(time.Minute) == 0:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return d.String()
	}
}
