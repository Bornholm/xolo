package model

import (
	"encoding/json"
	"time"
)

type BillingMode string

const (
	BillingModePayg         BillingMode = "payg"
	BillingModeSubscription BillingMode = "subscription"
)

type PlanConstraintKind string

const (
	ConstraintRollingWindow PlanConstraintKind = "rolling_window"
	ConstraintConcurrency   PlanConstraintKind = "concurrency"
)

// PlanDuration wraps time.Duration with human-readable JSON marshaling ("5h", "168h", "30m", etc.).
type PlanDuration time.Duration

func (d PlanDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *PlanDuration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = PlanDuration(dur)
	return nil
}

func (d PlanDuration) Duration() time.Duration { return time.Duration(d) }

// PlanConstraint describes a single limit within a subscription plan.
// Fields used depend on Kind:
//   - rolling_window: Duration (e.g. "5h", "168h"), TokenBudget (optional), ValueBudget (optional, microcents in provider currency)
//   - concurrency:    MaxConcurrent
type PlanConstraint struct {
	Kind          PlanConstraintKind `json:"kind"`
	Label         string             `json:"label"`
	Duration      PlanDuration       `json:"duration,omitempty"`
	TokenBudget   *int64             `json:"token_budget,omitempty"`
	ValueBudget   *int64             `json:"value_budget,omitempty"` // microcents in provider currency
	MaxConcurrent *int               `json:"max_concurrent,omitempty"`
	// WindowAnchor aligns a rolling_window constraint on a fixed (tumbling) schedule.
	// It records any instant at which a window opened; combined with Duration it lets us
	// compute the current window boundaries so they match the upstream provider's real
	// reset schedule (e.g. MiniMax "Resets in 4h29"). When nil, the window behaves as a
	// continuous sliding window (now - Duration).
	WindowAnchor *time.Time `json:"window_anchor,omitempty"`
}

// IsAnchored reports whether the constraint uses a fixed (tumbling) window aligned on a
// manual anchor, as opposed to a continuous sliding window.
func (c PlanConstraint) IsAnchored() bool {
	return c.WindowAnchor != nil && c.Duration.Duration() > 0
}

// CurrentWindowStart returns the start instant of the tumbling window containing `now`,
// aligned on WindowAnchor. When no anchor is set it falls back to a sliding window
// (now - Duration). Returns the zero time when Duration is not positive.
func (c PlanConstraint) CurrentWindowStart(now time.Time) time.Time {
	dur := c.Duration.Duration()
	if dur <= 0 {
		return time.Time{}
	}
	if c.WindowAnchor == nil {
		return now.Add(-dur)
	}
	anchor := *c.WindowAnchor
	elapsed := now.Sub(anchor)
	n := elapsed / dur
	if elapsed < 0 && elapsed%dur != 0 {
		n-- // floor toward negative infinity
	}
	return anchor.Add(n * dur)
}

// NextResetAt returns the instant at which the current tumbling window resets. It is only
// meaningful for anchored windows; sliding windows never reset atomically, so it returns
// the zero time when no anchor is set.
func (c PlanConstraint) NextResetAt(now time.Time) time.Time {
	if !c.IsAnchored() {
		return time.Time{}
	}
	ws := c.CurrentWindowStart(now)
	if ws.IsZero() {
		return time.Time{}
	}
	return ws.Add(c.Duration.Duration())
}

// SubscriptionPlan describes the limits attached to a subscription-billed provider.
type SubscriptionPlan struct {
	Label       string           `json:"label"`
	Constraints []PlanConstraint `json:"constraints"`
}
