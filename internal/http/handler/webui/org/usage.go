package org

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
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
)

const usagePageSize = 20

func (h *Handler) getUsagePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	offset := (page - 1) * usagePageSize

	rangeParam := r.URL.Query().Get("range")
	since := rangeToSince(rangeParam)
	orgID := org.ID()

	orgCurrency := org.Currency()

	// Aggregate without currency filter so all records are counted
	agg, err := h.usageStore.AggregateUsage(ctx, port.UsageFilter{
		OrgID: &orgID,
		Since: &since,
	})
	if err != nil {
		slog.ErrorContext(ctx, "could not aggregate usage", slogx.Error(err))
		agg = &port.UsageAggregate{}
	}

	// Compute total cost with per-currency conversion
	if agg != nil {
		byCurrency, sumErr := h.usageStore.SumCostSinceByCurrency(ctx, nil, orgID, since)
		if sumErr != nil {
			slog.ErrorContext(ctx, "could not sum cost by currency", slogx.Error(sumErr))
		} else {
			var totalConverted int64
			for cur, amount := range byCurrency {
				converted, convErr := h.exchangeRateService.Convert(ctx, amount, cur, orgCurrency)
				if convErr != nil {
					slog.WarnContext(ctx, "currency conversion failed, using raw amount",
						slog.String("from", cur), slog.String("to", orgCurrency), slogx.Error(convErr))
					totalConverted += amount
				} else {
					totalConverted += converted
				}
			}
			agg.TotalCost = totalConverted
			agg.Currency = orgCurrency
		}
	}

	// Fetch one extra record to detect whether a next page exists
	rawRecords, err := h.usageStore.QueryUsage(ctx, port.UsageFilter{
		OrgID:  &orgID,
		Since:  &since,
		Limit:  intPtr(usagePageSize + 1),
		Offset: intPtr(offset),
	})
	if err != nil {
		slog.ErrorContext(ctx, "could not query usage records", slogx.Error(err))
		rawRecords = nil
	}

	hasNext := len(rawRecords) > usagePageSize
	if hasNext {
		rawRecords = rawRecords[:usagePageSize]
	}

	// Batch-load users for the records on this page
	users := make(map[model.UserID]model.User)
	for _, rec := range rawRecords {
		uid := rec.UserID()
		if _, ok := users[uid]; ok {
			continue
		}
		u, err := h.userStore.GetUserByID(ctx, uid)
		if err != nil {
			slog.WarnContext(ctx, "could not fetch user for usage record", slogx.Error(err), slog.String("userID", string(uid)))
			continue
		}
		users[uid] = u
	}

	// Pre-load models and providers for energy estimation (deduplicated)
	modelCache := make(map[model.LLMModelID]model.LLMModel)
	providerCache := make(map[model.ProviderID]model.Provider)
	for _, rec := range rawRecords {
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
	records := make([]component.OrgDisplayUsageRecord, 0, len(rawRecords))
	var totalEnergyWh, totalCO2GramsMid float64
	for _, rec := range rawRecords {
		displayModelName := rec.ProxyModelName()
		if rec.ResolvedModelName() != "" && rec.ResolvedModelName() != rec.ProxyModelName() {
			displayModelName = rec.ProxyModelName() + " → " + rec.ResolvedModelName()
		}
		dr := component.OrgDisplayUsageRecord{
			Record:           rec,
			DisplayModelName: displayModelName,
			DisplayCost:      rec.Cost(),
			DisplayCurrency:  rec.Currency(),
		}
		if orgCurrency != rec.Currency() {
			converted, convErr := h.exchangeRateService.Convert(ctx, rec.Cost(), rec.Currency(), orgCurrency)
			if convErr == nil {
				dr.DisplayCost = converted
				dr.DisplayCurrency = orgCurrency
				dr.Converted = true
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
		records = append(records, dr)
	}

	// Fetch org quota for budget pie charts
	var orgQuota model.Quota
	if quotaStore, ok := h.orgStore.(port.QuotaStore); ok {
		if q, err := quotaStore.GetQuota(ctx, model.QuotaScopeOrg, string(orgID)); err == nil && q != nil {
			orgQuota = q
		}
	}

	// Compute daily/monthly/yearly spending for quota pie charts
	now := time.Now()
	dailyCost := h.sumConvertedCost(ctx, orgID, startOfPeriod("day", now), orgCurrency)
	monthlyCost := h.sumConvertedCost(ctx, orgID, startOfPeriod("month", now), orgCurrency)
	yearlyCost := h.sumConvertedCost(ctx, orgID, startOfPeriod("year", now), orgCurrency)

	// Fetch all records (up to 500) for chart aggregation — separate from paged display records
	chartRecords, _ := h.usageStore.QueryUsage(ctx, port.UsageFilter{
		OrgID: &orgID,
		Since: &since,
		Limit: intPtr(500),
	})

	// Also load users referenced by chart records that may not be in the paged set
	for _, rec := range chartRecords {
		uid := rec.UserID()
		if _, ok := users[uid]; ok {
			continue
		}
		u, err := h.userStore.GetUserByID(ctx, uid)
		if err != nil {
			continue
		}
		users[uid] = u
	}

	// Build per-model, per-user, per-day aggregates (cost in org currency)
	perModel := make(map[string]int64)
	perUser := make(map[string]int64)
	perDay := make(map[string]int64)
	for _, rec := range chartRecords {
		cost := rec.Cost()
		if orgCurrency != rec.Currency() {
			if converted, convErr := h.exchangeRateService.Convert(ctx, cost, rec.Currency(), orgCurrency); convErr == nil {
				cost = converted
			}
		}
		perModel[rec.ProxyModelName()] += cost
		uid := rec.UserID()
		userName := string(uid)
		if u, ok := users[uid]; ok {
			userName = u.DisplayName()
		}
		perUser[userName] += cost
		perDay[rec.CreatedAt().Format("2006-01-02")] += cost
	}

	vmodel := component.OrgUsagePageVModel{
		Org:              org,
		Aggregate:        agg,
		Records:          records,
		Users:            users,
		Since:            since,
		Range:            rangeParam,
		Page:             page,
		PageSize:         usagePageSize,
		HasNext:          hasNext,
		OrgQuota:         orgQuota,
		DailyCost:        dailyCost,
		MonthlyCost:      monthlyCost,
		YearlyCost:       yearlyCost,
		Currency:         orgCurrency,
		ChartPerDay:      chartByDate(perDay),
		ChartPerModel:    chartByValue(perModel),
		ChartPerUser:     chartByValue(perUser),
		TotalEnergyWh:    totalEnergyWh,
		TotalCO2GramsMid: totalCO2GramsMid,
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "org-" + orgSlug + "-usage",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Usage", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.OrgUsagePage(vmodel)).ServeHTTP(w, r)
}

func intPtr(n int) *int { return &n }

func rangeToSince(r string) time.Time {
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
	default: // "7d" and anything else
		return now.AddDate(0, 0, -7)
	}
}

// sumConvertedCost sums org costs from the given time, converting each currency to targetCurrency.
func (h *Handler) sumConvertedCost(ctx context.Context, orgID model.OrgID, since time.Time, targetCurrency string) int64 {
	byCurrency, err := h.usageStore.SumCostSinceByCurrency(ctx, nil, orgID, since)
	if err != nil {
		return 0
	}
	var total int64
	for cur, amount := range byCurrency {
		converted, err := h.exchangeRateService.Convert(ctx, amount, cur, targetCurrency)
		if err != nil {
			total += amount
		} else {
			total += converted
		}
	}
	return total
}

func startOfPeriod(period string, t time.Time) time.Time {
	y, m, d := t.Date()
	switch period {
	case "day":
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	case "month":
		return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
	default: // year
		return time.Date(y, 1, 1, 0, 0, 0, 0, t.Location())
	}
}

func chartByValue(m map[string]int64) []component.ChartDataPoint {
	pts := make([]component.ChartDataPoint, 0, len(m))
	for label, cost := range m {
		pts = append(pts, component.ChartDataPoint{Label: label, Value: float64(cost) / 1_000_000})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].Value > pts[j].Value })
	return pts
}

func chartByDate(m map[string]int64) []component.ChartDataPoint {
	dates := make([]string, 0, len(m))
	for k := range m {
		dates = append(dates, k)
	}
	sort.Strings(dates)
	pts := make([]component.ChartDataPoint, 0, len(dates))
	for _, date := range dates {
		pts = append(pts, component.ChartDataPoint{Label: date, Value: float64(m[date]) / 1_000_000})
	}
	return pts
}
