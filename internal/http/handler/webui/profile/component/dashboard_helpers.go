package component

import (
	"fmt"
	"strings"

	"github.com/xolo-gateway/xolo/internal/core/port"
	common "github.com/xolo-gateway/xolo/internal/http/handler/webui/common/component"
)

// personalDashboardMeta builds the grey line under the title: what the figures
// cover, the way the org dashboard states its own scope.
func personalDashboardMeta(vmodel DashboardPageVModel) string {
	parts := []string{"Vos appels"}

	if n := len(vmodel.OrgUsages); n > 0 {
		if n == 1 {
			parts = append(parts, "1 organisation")
		} else {
			parts = append(parts, fmt.Sprintf("%d organisations", n))
		}
	}

	if vmodel.Aggregate != nil && vmodel.Aggregate.Currency != "" {
		parts = append(parts, "devise "+vmodel.Aggregate.Currency)
	}

	if label := common.RangeLabel(vmodel.Range); label != "" {
		parts = append(parts, label)
	}

	return strings.Join(parts, " · ")
}

// personalCachedTokensNote words the share of prompt tokens served from the
// provider cache, under the Tokens figure of the KPI band. It returns the empty
// string when nothing was cached, so the cell carries no note rather than a
// "0 % en cache" that says nothing.
//
// It mirrors the org dashboard's cachedTokensNote: the two screens read the same
// aggregate and must word it identically, but they live in different packages.
func personalCachedTokensNote(agg *port.UsageAggregate) string {
	if agg == nil || agg.CachedTokens <= 0 || agg.PromptTokens <= 0 {
		return ""
	}

	return fmt.Sprintf("Dont %s en cache (%.0f %% des tokens prompt)",
		common.FormatCount(agg.CachedTokens),
		float64(agg.CachedTokens)/float64(agg.PromptTokens)*100,
	)
}

// personalTotalCostValue renders the total cost, or an em dash when nothing was
// billed — a zero would read as a measured value rather than as an absence.
func personalTotalCostValue(vmodel DashboardPageVModel) string {
	if vmodel.Aggregate == nil || vmodel.Aggregate.TotalCost == 0 {
		return "—"
	}

	return common.FormatCostCompact(vmodel.Aggregate.TotalCost, vmodel.Aggregate.Currency)
}

// personalRequests and personalTokens read the aggregate defensively: the KPI
// band is rendered on every visit, including the one where nothing has been
// consumed yet and the handler has no aggregate to hand over.
func personalRequests(agg *port.UsageAggregate) int64 {
	if agg == nil {
		return 0
	}
	return agg.TotalRequests
}

func personalTokens(agg *port.UsageAggregate) int64 {
	if agg == nil {
		return 0
	}
	return agg.TotalTokens
}

// personalCurrency names the currency the figures are expressed in. It labels
// the cost series and formats the per-provider ranking, both of which read the
// same aggregate.
func personalCurrency(vmodel DashboardPageVModel) string {
	if vmodel.Aggregate != nil && vmodel.Aggregate.Currency != "" {
		return vmodel.Aggregate.Currency
	}
	return "EUR"
}

// personalRecordsRangeLabel numbers the rows of the current page, the way the
// org dashboard does above its own table.
func personalRecordsRangeLabel(vmodel DashboardPageVModel) string {
	if len(vmodel.Records) == 0 {
		return ""
	}

	page := vmodel.Page
	if page < 1 {
		page = 1
	}

	size := vmodel.PageSize
	if size < 1 {
		size = len(vmodel.Records)
	}

	first := (page-1)*size + 1

	return fmt.Sprintf("%d – %d", first, first+len(vmodel.Records)-1)
}

// personalOrgBudgetVModel is one line of « Mon budget par organisation »: which
// organisation, on which of its budgets, and how far along it the user is.
type personalOrgBudgetVModel struct {
	Name        string
	Roles       string
	Budget      *int64
	Spent       int64
	PeriodLabel string
	Percent     int
	Currency    string
}

func newPersonalOrgBudget(ou OrgUsage) personalOrgBudgetVModel {
	vm := personalOrgBudgetVModel{
		Name:        string(ou.Membership.OrgID()),
		PeriodLabel: "mensuel",
		Currency:    ou.Currency,
	}
	if ou.Membership.Org() != nil {
		vm.Name = ou.Membership.Org().Name()
	}

	names := make([]string, 0, len(ou.Membership.Roles()))
	for _, role := range ou.Membership.Roles() {
		names = append(names, role.Name())
	}
	vm.Roles = strings.Join(names, ", ")

	switch {
	case ou.Quota != nil && ou.Quota.MonthlyBudget != nil:
		vm.Budget, vm.Spent, vm.PeriodLabel = ou.Quota.MonthlyBudget, ou.MonthlyCost, "mensuel"
	case ou.Quota != nil && ou.Quota.YearlyBudget != nil:
		vm.Budget, vm.Spent, vm.PeriodLabel = ou.Quota.YearlyBudget, ou.YearlyCost, "annuel"
	case ou.Quota != nil && ou.Quota.DailyBudget != nil:
		vm.Budget, vm.Spent, vm.PeriodLabel = ou.Quota.DailyBudget, ou.DailyCost, "journalier"
	default:
		vm.Spent = ou.MonthlyCost
	}
	vm.Percent = common.UsagePercent(vm.Spent, vm.Budget)

	return vm
}

// SpentLabel words the right end of the line: consumed over ceiling, or the
// consumption alone when the organisation set no ceiling.
func (vm personalOrgBudgetVModel) SpentLabel() string {
	spent := common.FormatCost(vm.Spent, vm.Currency)
	if vm.Budget == nil {
		return spent + " " + periodAdverb(vm.PeriodLabel)
	}
	return fmt.Sprintf("%s / %s %s", spent, common.FormatCost(*vm.Budget, vm.Currency), periodAdverb(vm.PeriodLabel))
}

// periodAdverb turns the adjective carried by the quota into the phrase the
// mockup puts after the figures ("8,20 € / 53,48 € ce mois").
func periodAdverb(periodLabel string) string {
	switch periodLabel {
	case "journalier":
		return "aujourd'hui"
	case "annuel":
		return "cette année"
	default:
		return "ce mois"
	}
}
