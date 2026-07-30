package component

import (
	"fmt"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	commonComp "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

// OverviewPageVModel backs the platform overview, the landing screen of the
// console.
type OverviewPageVModel struct {
	AppLayoutVModel commonComp.AppLayoutVModel

	// Currency is the reference every figure of the screen is converted into,
	// since organisations may each bill in a different one.
	Currency string
	// Since is the start of the reporting window.
	Since time.Time

	TotalOrgs     int
	TotalMembers  int
	TotalCost     int64
	TotalTokens   int64
	TotalRequests int64

	Orgs []OverviewOrg
	Days []OverviewDay
}

// OverviewOrg is one organisation of the overview, with its usage over the
// window.
type OverviewOrg struct {
	ID       model.OrgID
	Name     string
	Slug     string
	Active   bool
	Members  int
	Cost     int64
	Tokens   int64
	Requests int64
	// ColorClass is the chart colour the organisation is drawn with, shared by
	// the histogram legend and its segments.
	ColorClass string
}

// OverviewDay is one column of the stacked cost histogram.
type OverviewDay struct {
	Label    string
	Total    int64
	Segments []OverviewSegment
}

// OverviewSegment is one organisation's share of a day.
type OverviewSegment struct {
	OrgName    string
	ColorClass string
	Cost       int64
	// HeightPct is the share of the tallest column of the chart, so all columns
	// share a scale.
	HeightPct float64
}

// overviewOrgColors is the fixed series order of the design system. Beyond five
// organisations the palette repeats: the chart is a shape, and the table right
// under it carries the exact figures.
var overviewOrgColors = []string{"bg-chart-1", "bg-chart-2", "bg-chart-3", "bg-chart-4", "bg-chart-5"}

// OverviewOrgColor returns the chart colour of the nth organisation.
func OverviewOrgColor(i int) string {
	return overviewOrgColors[i%len(overviewOrgColors)]
}

// overviewSubtitle states the scope of the figures, as the mockup does.
func overviewSubtitle(vmodel OverviewPageVModel) string {
	return fmt.Sprintf(
		"Consommation agrégée de %d organisation(s) · %d membre(s) · 30 derniers jours · devise de référence %s",
		vmodel.TotalOrgs, vmodel.TotalMembers, vmodel.Currency,
	)
}

// overviewAxisLabels picks the few dates written under the histogram: the first,
// the last, and two in between — more would collide at this column width.
func overviewAxisLabels(days []OverviewDay) []string {
	if len(days) == 0 {
		return nil
	}
	if len(days) <= 4 {
		labels := make([]string, 0, len(days))
		for _, d := range days {
			labels = append(labels, d.Label)
		}
		return labels
	}
	last := len(days) - 1
	return []string{
		days[0].Label,
		days[last/3].Label,
		days[2*last/3].Label,
		days[last].Label,
	}
}
