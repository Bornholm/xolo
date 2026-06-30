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
}

// SubscriptionPlan describes the limits attached to a subscription-billed provider.
type SubscriptionPlan struct {
	Label       string           `json:"label"`
	Constraints []PlanConstraint `json:"constraints"`
}
