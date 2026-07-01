package model

import (
	"testing"
	"time"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return v
}

func TestPlanConstraint_CurrentWindowStart_Sliding(t *testing.T) {
	c := PlanConstraint{Kind: ConstraintRollingWindow, Duration: PlanDuration(5 * time.Hour)}
	now := mustTime(t, "2026-07-01T12:30:00Z")

	if got := c.CurrentWindowStart(now); !got.Equal(now.Add(-5 * time.Hour)) {
		t.Fatalf("sliding window start = %s, want %s", got, now.Add(-5*time.Hour))
	}
	if c.IsAnchored() {
		t.Fatal("constraint without anchor must not be anchored")
	}
	if got := c.NextResetAt(now); !got.IsZero() {
		t.Fatalf("sliding window NextResetAt = %s, want zero", got)
	}
}

func TestPlanConstraint_CurrentWindowStart_Anchored(t *testing.T) {
	anchor := mustTime(t, "2026-07-01T05:00:00Z")
	c := PlanConstraint{
		Kind:         ConstraintRollingWindow,
		Duration:     PlanDuration(5 * time.Hour),
		WindowAnchor: &anchor,
	}

	cases := []struct {
		now       string
		wantStart string
		wantReset string
	}{
		// Same window as the anchor.
		{"2026-07-01T05:00:00Z", "2026-07-01T05:00:00Z", "2026-07-01T10:00:00Z"},
		{"2026-07-01T09:59:59Z", "2026-07-01T05:00:00Z", "2026-07-01T10:00:00Z"},
		// Next window after one full period.
		{"2026-07-01T10:00:00Z", "2026-07-01T10:00:00Z", "2026-07-01T15:00:00Z"},
		{"2026-07-01T12:30:00Z", "2026-07-01T10:00:00Z", "2026-07-01T15:00:00Z"},
		// Several periods later: anchor 05:00 + 4×5h = 01:00 next day (exact boundary).
		{"2026-07-02T01:00:00Z", "2026-07-02T01:00:00Z", "2026-07-02T06:00:00Z"},
		{"2026-07-02T03:30:00Z", "2026-07-02T01:00:00Z", "2026-07-02T06:00:00Z"},
		// Before the anchor (floor toward negative infinity).
		{"2026-07-01T04:59:59Z", "2026-07-01T00:00:00Z", "2026-07-01T05:00:00Z"},
	}

	if !c.IsAnchored() {
		t.Fatal("anchored constraint must report IsAnchored()")
	}

	for _, tc := range cases {
		now := mustTime(t, tc.now)
		if got := c.CurrentWindowStart(now); !got.Equal(mustTime(t, tc.wantStart)) {
			t.Errorf("now=%s: window start = %s, want %s", tc.now, got, tc.wantStart)
		}
		if got := c.NextResetAt(now); !got.Equal(mustTime(t, tc.wantReset)) {
			t.Errorf("now=%s: reset = %s, want %s", tc.now, got, tc.wantReset)
		}
	}
}
