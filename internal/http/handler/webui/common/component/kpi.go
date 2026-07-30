package component

import (
	"fmt"
	"strings"
)

// KPIProps is one cell of a KPI band.
type KPIProps struct {
	Label string
	Value string
	// Delta is the signed variation against the previous period ("+18,4 %").
	Delta     string
	DeltaTone Tone
	// ValueTone colours the figure itself. Used for the rates the design system
	// wants read as an alert (error rate above its SLO).
	ValueTone Tone
	// Note is the small grey line closing the cell ("SLO 1,0 % · dépassé").
	Note string
	// Spark holds the values of the optional sparkline. Fewer than two points
	// renders nothing.
	Spark []float64
	// SparkClass is the stroke utility of the sparkline ("text-chart-1").
	SparkClass string
}

// SparkPoints turns a series into the `points` attribute of an SVG polyline
// drawn in the 100×24 viewBox of the KPI sparkline.
//
// The series is normalised to its own min/max: a sparkline carries a shape, not
// a scale, and the figure right above it carries the magnitude. A flat series
// is drawn on the middle line rather than collapsed onto an edge.
func SparkPoints(values []float64) string {
	if len(values) < 2 {
		return ""
	}

	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	span := max - min
	var points strings.Builder
	for i, v := range values {
		x := float64(i) / float64(len(values)-1) * 100
		// 2px of headroom top and bottom so the stroke is not clipped.
		y := 12.0
		if span > 0 {
			y = 22 - (v-min)/span*20
		}
		if i > 0 {
			points.WriteByte(' ')
		}
		fmt.Fprintf(&points, "%.2f,%.2f", x, y)
	}
	return points.String()
}
