package component

import (
	"testing"
	"time"
)

func TestCostSeriesFillsEveryDayOfThePeriod(t *testing.T) {
	since := time.Date(2026, 6, 1, 9, 30, 0, 0, time.Local)
	until := time.Date(2026, 6, 5, 18, 0, 0, 0, time.Local)

	pts := CostSeries(map[string]int64{
		"2026-06-01": 1_000_000,
		"2026-06-04": 2_400,
	}, since, until, "7d")

	if len(pts) != 5 {
		t.Fatalf("a five-day window should draw five bars, got %d: %+v", len(pts), pts)
	}
	if pts[0].Label != "1 juin" || pts[0].Value != 1 {
		t.Errorf("first bar: got %+v", pts[0])
	}
	if pts[1].Value != 0 || pts[2].Value != 0 {
		t.Errorf("the quiet days should be charted at zero, got %+v and %+v", pts[1], pts[2])
	}
	if pts[3].Value != 0.0024 {
		t.Errorf("micro-units should be divided by a million, got %v", pts[3].Value)
	}
}

func TestCostSeriesAggregatesLongPeriodsByMonth(t *testing.T) {
	since := time.Date(2025, 8, 15, 0, 0, 0, 0, time.Local)
	until := time.Date(2026, 1, 20, 0, 0, 0, 0, time.Local)

	pts := CostSeries(map[string]int64{
		"2025-08-20": 1_000_000,
		"2025-08-31": 500_000,
		"2025-12-02": 3_000_000,
	}, since, until, "365d")

	if len(pts) != 6 {
		t.Fatalf("august to january is six monthly bars, got %d: %+v", len(pts), pts)
	}
	if pts[0].Label != "août 25" {
		t.Errorf("monthly labels carry the year: got %q", pts[0].Label)
	}
	if pts[0].Value != 1.5 {
		t.Errorf("the days of a month should be summed into it, got %v", pts[0].Value)
	}
	if pts[4].Label != "déc. 25" || pts[4].Value != 3 {
		t.Errorf("december: got %+v", pts[4])
	}
	if pts[5].Value != 0 {
		t.Errorf("a month without usage stays on the axis at zero, got %+v", pts[5])
	}
}

// The chart sits directly under the total of the same period: whatever the
// bucketing, the bars must add up to it.
func TestCostSeriesKeepsTheTotal(t *testing.T) {
	perDay := map[string]int64{
		"2026-01-03": 120_000,
		"2026-02-17": 4_000,
		"2026-05-30": 76_000,
	}
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	until := time.Date(2026, 6, 30, 0, 0, 0, 0, time.Local)

	for _, r := range []string{"30d", "365d"} {
		var sum float64
		for _, p := range CostSeries(perDay, since, until, r) {
			sum += p.Value
		}
		if sum != 0.2 {
			t.Errorf("range %q: bars sum to %v, expected the period total 0.2", r, sum)
		}
	}
}

// Usage dated outside the window still belongs to the total displayed above the
// chart, so it must not be silently dropped.
func TestCostSeriesKeepsUsageOutsideTheWindow(t *testing.T) {
	since := time.Date(2026, 6, 10, 0, 0, 0, 0, time.Local)
	until := time.Date(2026, 6, 12, 0, 0, 0, 0, time.Local)

	pts := CostSeries(map[string]int64{"2026-06-02": 5_000_000}, since, until, "7d")

	var sum float64
	for _, p := range pts {
		sum += p.Value
	}
	if sum != 5 {
		t.Errorf("stray day dropped: bars sum to %v", sum)
	}
}

func TestRangeBucket(t *testing.T) {
	for r, want := range map[string]SeriesBucket{
		"":     SeriesBucketDay,
		"1d":   SeriesBucketDay,
		"7d":   SeriesBucketDay,
		"30d":  SeriesBucketDay,
		"90d":  SeriesBucketMonth,
		"180d": SeriesBucketMonth,
		"365d": SeriesBucketMonth,
	} {
		if got := RangeBucket(r); got != want {
			t.Errorf("range %q: got %q, want %q", r, got, want)
		}
	}
	if got := CostSeriesTitle("365d"); got != "Coût par mois" {
		t.Errorf("title on a yearly range: %q", got)
	}
	if got := CostSeriesTitle("7d"); got != "Coût par jour" {
		t.Errorf("title on a weekly range: %q", got)
	}
}
