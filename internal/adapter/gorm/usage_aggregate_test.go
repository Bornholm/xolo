package gorm_test

import (
	"context"
	"testing"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// recordUsage is a small helper building and storing a usage record. The gorm layer
// auto-populates created_at with the current time, so all records land in "today".
func recordUsage(t *testing.T, store interface {
	RecordUsage(context.Context, model.UsageRecord) error
}, userID, provider, modelID, proxyName, resolvedName, currency string, cost int64, promptTokens, completionTokens int, planCovered bool) {
	t.Helper()
	rec := model.NewUsageRecord(
		model.UserID(userID), "", "org-1", model.ProviderID(provider), model.LLMModelID(modelID),
		proxyName, "", promptTokens, 0, completionTokens, cost, currency, model.CostSourceProvider, resolvedName,
	)
	rec.SetPlanCovered(planCovered)
	if err := store.RecordUsage(context.Background(), rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
}

func costByKey(rows []port.DimensionCost) map[string]int64 {
	out := make(map[string]int64)
	for _, r := range rows {
		out[r.Key] += r.Cost
	}
	return out
}

func TestAggregateCostByDimension(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	orgID := model.OrgID("org-1")

	// PAYG records.
	recordUsage(t, store, "u1", "p1", "m-gpt", "gpt-4", "", "USD", 1000, 10, 20, false)
	recordUsage(t, store, "u1", "p1", "m-gpt", "virtual", "gpt-4", "USD", 2000, 10, 20, false)
	recordUsage(t, store, "u2", "p2", "m-claude", "claude", "", "EUR", 500, 5, 5, false)
	// Subscription-covered record: excluded from every cost aggregation.
	recordUsage(t, store, "u1", "p1", "m-gpt", "gpt-4", "", "USD", 999999, 40, 60, true)

	since := time.Now().Add(-time.Hour)
	filter := port.UsageFilter{OrgID: &orgID, Since: &since}

	t.Run("by model (effective name, plan excluded)", func(t *testing.T) {
		rows, err := store.AggregateCostByDimension(ctx, filter, port.UsageDimensionModel)
		if err != nil {
			t.Fatalf("AggregateCostByDimension: %v", err)
		}
		got := costByKey(rows)
		if got["gpt-4"] != 3000 {
			t.Errorf("gpt-4: expected 3000, got %d", got["gpt-4"])
		}
		if got["claude"] != 500 {
			t.Errorf("claude: expected 500, got %d", got["claude"])
		}
		if len(got) != 2 {
			t.Errorf("expected 2 model buckets, got %d (%v)", len(got), got)
		}
		// Org and currency must be surfaced for per-org conversion.
		for _, r := range rows {
			if r.OrgID != orgID {
				t.Errorf("unexpected org %q", r.OrgID)
			}
			if r.Currency == "" {
				t.Errorf("missing currency for key %q", r.Key)
			}
		}
	})

	t.Run("by user", func(t *testing.T) {
		rows, err := store.AggregateCostByDimension(ctx, filter, port.UsageDimensionUser)
		if err != nil {
			t.Fatalf("AggregateCostByDimension: %v", err)
		}
		got := costByKey(rows)
		if got["u1"] != 3000 {
			t.Errorf("u1: expected 3000 (plan-covered excluded), got %d", got["u1"])
		}
		if got["u2"] != 500 {
			t.Errorf("u2: expected 500, got %d", got["u2"])
		}
	})

	t.Run("by provider", func(t *testing.T) {
		rows, err := store.AggregateCostByDimension(ctx, filter, port.UsageDimensionProvider)
		if err != nil {
			t.Fatalf("AggregateCostByDimension: %v", err)
		}
		got := costByKey(rows)
		if got["p1"] != 3000 {
			t.Errorf("p1: expected 3000, got %d", got["p1"])
		}
		if got["p2"] != 500 {
			t.Errorf("p2: expected 500, got %d", got["p2"])
		}
	})

	t.Run("by day keeps currencies separate", func(t *testing.T) {
		rows, err := store.AggregateCostByDimension(ctx, filter, port.UsageDimensionDay)
		if err != nil {
			t.Fatalf("AggregateCostByDimension: %v", err)
		}
		// All records are "today" but in two currencies → two rows, one date bucket.
		var totalUSD, totalEUR int64
		dates := map[string]struct{}{}
		for _, r := range rows {
			dates[r.Key] = struct{}{}
			switch r.Currency {
			case "USD":
				totalUSD += r.Cost
			case "EUR":
				totalEUR += r.Cost
			}
		}
		if len(dates) != 1 {
			t.Errorf("expected records in a single day bucket, got %d", len(dates))
		}
		if totalUSD != 3000 || totalEUR != 500 {
			t.Errorf("expected USD=3000 EUR=500, got USD=%d EUR=%d", totalUSD, totalEUR)
		}
	})
}

func TestAggregatePlanTokensByUser(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	orgID := model.OrgID("org-1")

	// Two plan-covered records for u1 (40+60 and 15+5 tokens) and a PAYG record that
	// must NOT be counted.
	recordUsage(t, store, "u1", "p1", "m-gpt", "gpt-4", "", "USD", 0, 40, 60, true)
	recordUsage(t, store, "u1", "p1", "m-gpt", "gpt-4", "", "USD", 0, 15, 5, true)
	recordUsage(t, store, "u1", "p1", "m-gpt", "gpt-4", "", "USD", 1000, 100, 100, false)
	recordUsage(t, store, "u2", "p1", "m-gpt", "gpt-4", "", "USD", 0, 1, 1, true)

	since := time.Now().Add(-time.Hour)
	filter := port.UsageFilter{OrgID: &orgID, Since: &since}

	rows, err := store.AggregatePlanTokensByUser(ctx, filter)
	if err != nil {
		t.Fatalf("AggregatePlanTokensByUser: %v", err)
	}
	got := make(map[model.UserID]int64)
	for _, r := range rows {
		got[r.UserID] = r.Tokens
	}
	// u1: (40+60) + (15+5) = 120 ; the PAYG 200-token record is excluded.
	if got["u1"] != 120 {
		t.Errorf("u1: expected 120 plan tokens, got %d", got["u1"])
	}
	if got["u2"] != 2 {
		t.Errorf("u2: expected 2 plan tokens, got %d", got["u2"])
	}
}
