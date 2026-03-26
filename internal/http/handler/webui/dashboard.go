package webui

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/estimator"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/profile/component"
)

const dashboardPageSize = 20

func (h *Handler) getDashboardPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	memberships := httpCtx.Memberships(ctx)
	baseURL := httpCtx.BaseURL(ctx)

	// No memberships → redirect to /no-org
	if len(memberships) == 0 {
		http.Redirect(w, r, baseURL.JoinPath("/no-org").String(), http.StatusTemporaryRedirect)
		return
	}

	// Pagination
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	offset := (page - 1) * dashboardPageSize

	// Time range
	rangeParam := r.URL.Query().Get("range")
	since := dashboardRangeToSince(rangeParam)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	// Build per-org quota usage
	userID := user.ID()
	orgUsages := make([]component.OrgUsage, 0, len(memberships))
	for _, m := range memberships {
		quota, err := h.quotaService.ResolveEffectiveQuota(ctx, userID, m.OrgID())
		if err != nil {
			slog.ErrorContext(ctx, "could not resolve quota", slogx.Error(err), slog.String("orgID", string(m.OrgID())))
		}

		currency := model.DefaultCurrency
		if quota != nil {
			currency = quota.Currency
		}

		daily, err := h.sumCostConverted(ctx, userID, m.OrgID(), currency, todayStart)
		if err != nil {
			slog.ErrorContext(ctx, "could not sum daily cost", slogx.Error(err))
		}
		monthly, err := h.sumCostConverted(ctx, userID, m.OrgID(), currency, monthStart)
		if err != nil {
			slog.ErrorContext(ctx, "could not sum monthly cost", slogx.Error(err))
		}
		yearly, err := h.sumCostConverted(ctx, userID, m.OrgID(), currency, yearStart)
		if err != nil {
			slog.ErrorContext(ctx, "could not sum yearly cost", slogx.Error(err))
		}

		orgUsages = append(orgUsages, component.OrgUsage{
			Membership:  m,
			Quota:       quota,
			DailyCost:   daily,
			MonthlyCost: monthly,
			YearlyCost:  yearly,
			Currency:    currency,
		})
	}

	// Aggregate usage for the selected period (no currency filter: counts all records)
	agg, err := h.usageStore.AggregateUsage(ctx, port.UsageFilter{
		UserID: &userID,
		Since:  &since,
	})
	if err != nil {
		slog.ErrorContext(ctx, "could not aggregate usage", slogx.Error(err))
		agg = &port.UsageAggregate{}
	}

	// Compute total cost with per-currency conversion so mixed-currency records
	// (e.g. USD stored when EUR conversion failed) are properly included.
	if agg != nil && len(orgUsages) > 0 {
		// Use the first org's currency as the display currency for the total.
		displayCurrency := orgUsages[0].Currency
		var totalConverted int64
		for _, ou := range orgUsages {
			orgCur := ou.Currency
			converted, convErr := h.sumCostConverted(ctx, userID, ou.Membership.OrgID(), orgCur, since)
			if convErr != nil {
				slog.ErrorContext(ctx, "could not compute converted total", slogx.Error(convErr))
				continue
			}
			// If orgs have different currencies convert to display currency.
			if orgCur != displayCurrency {
				converted, convErr = h.exchangeRateService.Convert(ctx, converted, orgCur, displayCurrency)
				if convErr != nil {
					slog.WarnContext(ctx, "could not convert org total to display currency", slogx.Error(convErr))
				}
			}
			totalConverted += converted
		}
		agg.TotalCost = totalConverted
		agg.Currency = displayCurrency
	}

	// Paginated records
	records, err := h.usageStore.QueryUsage(ctx, port.UsageFilter{
		UserID: &userID,
		Since:  &since,
		Limit:  intPtr(dashboardPageSize + 1),
		Offset: intPtr(offset),
	})
	if err != nil {
		slog.ErrorContext(ctx, "could not query usage records", slogx.Error(err))
		records = nil
	}

	hasNext := len(records) > dashboardPageSize
	if hasNext {
		records = records[:dashboardPageSize]
	}

	// Build org name map for the table
	orgs := make(map[model.OrgID]model.Organization)
	for _, m := range memberships {
		if m.Org() != nil {
			orgs[m.OrgID()] = m.Org()
		}
	}

	// Pre-load models and providers for energy estimation (deduplicated)
	modelCache := make(map[model.LLMModelID]model.LLMModel)
	providerCache := make(map[model.ProviderID]model.Provider)
	for _, rec := range records {
		mid := rec.ModelID()
		if mid == "" {
			continue
		}
		if _, ok := modelCache[mid]; ok {
			continue
		}
		m, err := h.providerStore.GetLLMModelByID(ctx, mid)
		if err != nil {
			continue
		}
		modelCache[mid] = m
		pid := m.ProviderID()
		if _, ok := providerCache[pid]; !ok {
			p, err := h.providerStore.GetProviderByID(ctx, pid)
			if err == nil {
				providerCache[pid] = p
			}
		}
	}

	// Build display records with cost converted to org currency where needed
	displayRecords := make([]component.DisplayUsageRecord, 0, len(records))
	var totalEnergyWh, totalCO2GramsMid float64
	for _, rec := range records {
		displayModelName := rec.ProxyModelName()
		if rec.ResolvedModelName() != "" && rec.ResolvedModelName() != rec.ProxyModelName() {
			displayModelName = rec.ProxyModelName() + " → " + rec.ResolvedModelName()
		}
		dr := component.DisplayUsageRecord{
			Record:           rec,
			DisplayModelName: displayModelName,
			DisplayCost:      rec.Cost(),
			DisplayCurrency:  rec.Currency(),
		}
		if org, ok := orgs[rec.OrgID()]; ok {
			orgCurrency := org.Currency()
			if orgCurrency == "" {
				orgCurrency = model.DefaultCurrency
			}
			if orgCurrency != rec.Currency() {
				converted, convErr := h.exchangeRateService.Convert(ctx, rec.Cost(), rec.Currency(), orgCurrency)
				if convErr == nil {
					dr.DisplayCost = converted
					dr.DisplayCurrency = orgCurrency
					dr.Converted = true
				}
			}
		}
		// Energy estimation
		if m, ok := modelCache[rec.ModelID()]; ok && m.ActiveParams() > 0 {
			tier := estimator.TierHyperscaler
			if p, ok := providerCache[m.ProviderID()]; ok {
				tier = estimator.CloudTier(p.CloudTier())
			}
			est := estimator.NewCloudEstimator(tier).EstimateFromParams(
				float64(m.ActiveParams()),
				estimator.InferenceRequest{
					InputTokens:  int(rec.PromptTokens()),
					OutputTokens: int(rec.CompletionTokens()),
				},
				m.TokensPerSecLow(),
				m.TokensPerSecHigh(),
			)
			dr.EnergyWh = est.Mid.TotalWh
			dr.EnergyLowWh = est.Low.TotalWh
			dr.EnergyHighWh = est.High.TotalWh
			dr.CO2GramsMid = est.Mid.Equivalences.CO2Grams
			dr.CO2GramsMin = est.Mid.Equivalences.CO2GramsMin
			dr.CO2GramsMax = est.Mid.Equivalences.CO2GramsMax
			totalEnergyWh += est.Mid.TotalWh
			totalCO2GramsMid += est.Mid.Equivalences.CO2Grams
		}
		displayRecords = append(displayRecords, dr)
	}

	// Fetch all records (up to 500) for chart aggregation
	chartRecords, _ := h.usageStore.QueryUsage(ctx, port.UsageFilter{
		UserID: &userID,
		Since:  &since,
		Limit:  intPtr(500),
	})

	// Aggregate per-day and per-model (cost converted to org currency when possible)
	perDay := make(map[string]int64)
	perModel := make(map[string]int64)
	for _, rec := range chartRecords {
		cost := rec.Cost()
		if org, ok := orgs[rec.OrgID()]; ok {
			orgCurrency := org.Currency()
			if orgCurrency == "" {
				orgCurrency = model.DefaultCurrency
			}
			if orgCurrency != rec.Currency() {
				if conv, convErr := h.exchangeRateService.Convert(ctx, cost, rec.Currency(), orgCurrency); convErr == nil {
					cost = conv
				}
			}
		}
		perDay[rec.CreatedAt().Format("2006-01-02")] += cost
		perModel[rec.ProxyModelName()] += cost
	}

	vmodel := component.DashboardPageVModel{
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "usage",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace de travail", Href: "/usage"},
				{Label: "Usage", Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
		OrgUsages:        orgUsages,
		Aggregate:        agg,
		Records:          displayRecords,
		Orgs:             orgs,
		Range:            rangeParam,
		Page:             page,
		HasNext:          hasNext,
		ChartPerDay:      dashChartByDate(perDay),
		ChartPerModel:    dashChartByValue(perModel),
		TotalEnergyWh:    totalEnergyWh,
		TotalCO2GramsMid: totalCO2GramsMid,
	}

	templ.Handler(component.DashboardPage(vmodel)).ServeHTTP(w, r)
}

func dashboardRangeToSince(r string) time.Time {
	now := time.Now()
	switch r {
	case "1d":
		return now.AddDate(0, 0, -1)
	case "30d":
		return now.AddDate(0, -1, 0)
	case "90d":
		return now.AddDate(0, -3, 0)
	case "180d":
		return now.AddDate(0, -6, 0)
	case "365d":
		return now.AddDate(-1, 0, 0)
	default: // "7d"
		return now.AddDate(0, 0, -7)
	}
}

func intPtr(n int) *int { return &n }

func dashChartByValue(m map[string]int64) []component.ProfileChartDataPoint {
	pts := make([]component.ProfileChartDataPoint, 0, len(m))
	for label, cost := range m {
		pts = append(pts, component.ProfileChartDataPoint{Label: label, Value: float64(cost) / 1_000_000})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].Value > pts[j].Value })
	return pts
}

func dashChartByDate(m map[string]int64) []component.ProfileChartDataPoint {
	dates := make([]string, 0, len(m))
	for k := range m {
		dates = append(dates, k)
	}
	sort.Strings(dates)
	pts := make([]component.ProfileChartDataPoint, 0, len(dates))
	for _, d := range dates {
		pts = append(pts, component.ProfileChartDataPoint{Label: d, Value: float64(m[d]) / 1_000_000})
	}
	return pts
}

// sumCostConverted returns the total cost for a user+org since the given time,
// converting each currency's sub-total to targetCurrency using the exchange
// rate service. Falls back to the raw amount when conversion is unavailable.
func (h *Handler) sumCostConverted(ctx context.Context, userID model.UserID, orgID model.OrgID, targetCurrency string, since time.Time) (int64, error) {
	byCurrency, err := h.usageStore.SumCostSinceByCurrency(ctx, []model.UserID{userID}, orgID, since)
	if err != nil {
		return 0, err
	}
	var total int64
	for cur, amount := range byCurrency {
		converted, convErr := h.exchangeRateService.Convert(ctx, amount, cur, targetCurrency)
		if convErr != nil {
			slog.WarnContext(ctx, "could not convert currency for display, using raw amount",
				slog.String("from", cur), slog.String("to", targetCurrency), slogx.Error(convErr))
			total += amount
		} else {
			total += converted
		}
	}
	return total, nil
}
