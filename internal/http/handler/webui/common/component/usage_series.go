package component

import (
	"fmt"
	"sort"
	"time"
)

// SeriesBucket is the granularity the cost chart of a period is drawn on.
type SeriesBucket string

const (
	// SeriesBucketDay charts one bar per calendar day.
	SeriesBucketDay SeriesBucket = "day"
	// SeriesBucketMonth charts one bar per calendar month.
	SeriesBucketMonth SeriesBucket = "month"
)

// RangeBucket returns the granularity a period is charted on: a day beyond a
// quarter would mean up to 365 bars, which stops being a shape and becomes a
// texture, so the long periods are charted by month.
func RangeBucket(r string) SeriesBucket {
	switch r {
	case "90d", "180d", "365d":
		return SeriesBucketMonth
	default:
		return SeriesBucketDay
	}
}

// CostSeriesTitle names the cost chart after the bucket it actually draws.
func CostSeriesTitle(r string) string {
	if RangeBucket(r) == SeriesBucketMonth {
		return "Coût par mois"
	}
	return "Coût par jour"
}

// CostSeries turns the per-day cost sub-totals of a period into a continuous
// series: one point per bucket between `since` and `until`, the empty ones
// included.
//
// The store only returns the days that carry usage. Charting those alone draws a
// period out of the days that happened to have traffic — two bars a month apart
// sit side by side, and a quiet week is indistinguishable from a week that was
// never in the window. Filling the gaps is what makes the horizontal axis a
// timeline again.
//
// `perDay` is keyed by calendar day, "YYYY-MM-DD", in micro-units of currency,
// as returned by AggregateCostByDimension on the day dimension.
func CostSeries(perDay map[string]int64, since, until time.Time, r string) []ChartDataPoint {
	if until.Before(since) {
		return nil
	}

	bucket := RangeBucket(r)
	totals := make(map[string]int64, len(perDay))
	for day, cost := range perDay {
		key, err := seriesKey(day, bucket)
		if err != nil {
			// A key the store did not produce: keep it out of the timeline rather
			// than guess where it belongs.
			continue
		}
		totals[key] += cost
	}

	pts := make([]ChartDataPoint, 0, len(totals)+1)
	seen := make(map[string]bool, len(totals))
	for cursor := truncate(since, bucket); !cursor.After(until); cursor = next(cursor, bucket) {
		key := cursor.Format(keyLayout(bucket))
		seen[key] = true
		pts = append(pts, ChartDataPoint{
			Label: seriesLabel(cursor, bucket),
			Value: float64(totals[key]) / 1_000_000,
		})
	}

	// Usage recorded outside the window — the filter and the bucketing round
	// differently near the edges — would otherwise vanish from a chart whose
	// total is displayed right above it.
	var strays []string
	for key := range totals {
		if !seen[key] {
			strays = append(strays, key)
		}
	}
	if len(strays) > 0 {
		sort.Strings(strays)
		extra := make([]ChartDataPoint, 0, len(strays))
		for _, key := range strays {
			at, err := time.ParseInLocation(keyLayout(bucket), key, time.Local)
			if err != nil {
				continue
			}
			extra = append(extra, ChartDataPoint{
				Label: seriesLabel(at, bucket),
				Value: float64(totals[key]) / 1_000_000,
			})
		}
		pts = append(extra, pts...)
	}

	return pts
}

// seriesKey moves a "YYYY-MM-DD" day onto the bucket it belongs to.
func seriesKey(day string, bucket SeriesBucket) (string, error) {
	at, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return "", fmt.Errorf("parse usage day %q: %w", day, err)
	}
	return at.Format(keyLayout(bucket)), nil
}

func keyLayout(bucket SeriesBucket) string {
	if bucket == SeriesBucketMonth {
		return "2006-01"
	}
	return "2006-01-02"
}

// truncate snaps a date to the start of its bucket, so the first bar of the
// chart is a whole day or a whole month rather than the instant the period
// happens to start at.
func truncate(at time.Time, bucket SeriesBucket) time.Time {
	at = at.In(time.Local)
	if bucket == SeriesBucketMonth {
		return time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.Local)
}

func next(at time.Time, bucket SeriesBucket) time.Time {
	if bucket == SeriesBucketMonth {
		return at.AddDate(0, 1, 0)
	}
	return at.AddDate(0, 0, 1)
}

// frenchMonths are the abbreviations the mockup labels its axes with ("30 juin",
// "7 juil."). Go formats month names in English only.
var frenchMonths = [...]string{
	"janv.", "févr.", "mars", "avr.", "mai", "juin",
	"juil.", "août", "sept.", "oct.", "nov.", "déc.",
}

// seriesLabel words one tick of the horizontal axis: "7 juil." for a day,
// "juil. 25" for a month — the year is carried because a yearly period shows the
// same month twice.
func seriesLabel(at time.Time, bucket SeriesBucket) string {
	month := frenchMonths[int(at.Month())-1]
	if bucket == SeriesBucketMonth {
		return fmt.Sprintf("%s %02d", month, at.Year()%100)
	}
	return fmt.Sprintf("%d %s", at.Day(), month)
}
