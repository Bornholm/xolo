package org

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/adapter/cache"
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

	// Support comma-separated values emitted by the SelectBox component (e.g. user=id1,id2)
	rawUserFilter := r.URL.Query()["user"]
	var userFilter []string
	for _, v := range rawUserFilter {
		for _, part := range strings.Split(v, ",") {
			if part != "" {
				userFilter = append(userFilter, part)
			}
		}
	}

	// CSV export
	if r.URL.Query().Get("format") == "csv" {
		h.serveOrgUsageCSV(w, r, org, userFilter, since)
		return
	}

	// Fetch org members for filter
	members, _, err := h.orgStore.ListOrgMembers(ctx, org.ID(), port.ListOrgMembersOptions{})
	if err != nil {
		slog.WarnContext(ctx, "could not list org members", slogx.Error(err))
		members = nil
	}

	orgCurrency := org.Currency()

	// Build user filter for queries
	var userIDs []model.UserID
	if len(userFilter) > 0 {
		for _, uid := range userFilter {
			userIDs = append(userIDs, model.UserID(uid))
		}
	}

	usageFilter := port.UsageFilter{
		OrgID: &orgID,
		Since: &since,
	}
	if len(userIDs) > 0 {
		usageFilter.UserIDs = userIDs
	}

	// Aggregate without currency filter so all records are counted
	agg, err := h.usageStore.AggregateUsage(ctx, usageFilter)
	if err != nil {
		slog.ErrorContext(ctx, "could not aggregate usage", slogx.Error(err))
		agg = &port.UsageAggregate{}
	}

	// Compute total cost with per-currency conversion
	if agg != nil {
		byCurrency, sumErr := h.usageStore.SumCostSinceByCurrency(ctx, userIDs, orgID, since)
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
		OrgID:   &orgID,
		Since:   &since,
		Limit:   intPtr(usagePageSize + 1),
		Offset:  intPtr(offset),
		UserIDs: userIDs,
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

	// Cache key for energy totals (includes user filter for uniqueness)
	energyCacheKey := fmt.Sprintf("org-usage:%s:%d:%v", orgID, since.Unix(), userIDs)

	// Fetch all records for chart aggregation and energy totals
	chartRecords, _ := h.usageStore.QueryUsage(ctx, port.UsageFilter{
		OrgID:   &orgID,
		Since:   &since,
		UserIDs: userIDs,
	})

	// Pre-load models and providers for energy estimation (deduplicated) from chartRecords
	modelCache := make(map[model.LLMModelID]model.LLMModel)
	providerCache := make(map[model.ProviderID]model.Provider)
	for _, rec := range chartRecords {
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

	// Calculate energy totals (from cache or fresh computation)
	var totalEnergyWh, totalCO2GramsMid float64
	if cached, ok := cache.EnergyEstimateCache.Get(energyCacheKey); ok {
		totalEnergyWh = cached.TotalEnergyWh
		totalCO2GramsMid = cached.TotalCO2GramsMid
	} else {
		for _, rec := range chartRecords {
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
				totalEnergyWh += est.Mid.TotalWh
				totalCO2GramsMid += est.Mid.Equivalences.CO2Grams
			}
		}
		cache.EnergyEstimateCache.Add(energyCacheKey, cache.EnergyTotals{
			TotalEnergyWh:    totalEnergyWh,
			TotalCO2GramsMid: totalCO2GramsMid,
		})
	}

	// Build display records with cost converted to org currency where needed
	records := make([]component.OrgDisplayUsageRecord, 0, len(rawRecords))
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

	// Build per-model, per-user, per-day, per-provider aggregates (cost in org currency)
	perModel := make(map[string]int64)
	perUser := make(map[string]int64)
	perDay := make(map[string]int64)
	perProvider := make(map[model.ProviderID]int64)
	for _, rec := range chartRecords {
		cost := rec.Cost()
		if orgCurrency != rec.Currency() {
			if converted, convErr := h.exchangeRateService.Convert(ctx, cost, rec.Currency(), orgCurrency); convErr == nil {
				cost = converted
			}
		}
		perProvider[rec.ProviderID()] += cost
		effectiveModel := rec.ProxyModelName()
		if rec.ResolvedModelName() != "" {
			effectiveModel = rec.ResolvedModelName()
		}
		perModel[effectiveModel] += cost
		uid := rec.UserID()
		userName := string(uid)
		if u, ok := users[uid]; ok {
			userName = u.DisplayName()
		}
		perUser[userName] += cost
		perDay[rec.CreatedAt().Format("2006-01-02")] += cost
	}

	// Build provider name map for chart labels
	providerNames := make(map[model.ProviderID]string)
	for pid := range perProvider {
		if p, err := h.providerStore.GetProviderByID(ctx, pid); err == nil {
			providerNames[pid] = p.Name()
		} else {
			providerNames[pid] = string(pid)
		}
	}

	vmodel := component.OrgUsagePageVModel{
		Org:              org,
		Aggregate:        agg,
		Records:          records,
		Users:            users,
		Members:          members,
		UserFilter:       userFilter,
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
		ChartPerProvider: chartByProvider(perProvider, providerNames),
		TotalEnergyWh:    totalEnergyWh,
		TotalCO2GramsMid: totalCO2GramsMid,
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "org-" + orgSlug + "-usage",
			HomeLink:      "/orgs/" + orgSlug,
			AdminSubtitle: "Admin. " + org.Name(),
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

func chartByProvider(m map[model.ProviderID]int64, names map[model.ProviderID]string) []component.ChartDataPoint {
	pts := make([]component.ChartDataPoint, 0, len(m))
	for pid, cost := range m {
		label := names[pid]
		if label == "" {
			label = string(pid)
		}
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

func (h *Handler) serveOrgUsageCSV(w http.ResponseWriter, r *http.Request, org model.Organization, userFilter []string, since time.Time) {
	ctx := r.Context()
	orgID := org.ID()

	var userIDs []model.UserID
	for _, uid := range userFilter {
		userIDs = append(userIDs, model.UserID(uid))
	}

	usageFilter := port.UsageFilter{
		OrgID: &orgID,
		Since: &since,
	}
	if len(userIDs) > 0 {
		usageFilter.UserIDs = userIDs
	}

	records, err := h.usageStore.QueryUsage(ctx, usageFilter)
	if err != nil {
		slog.ErrorContext(ctx, "could not query usage records for CSV", slogx.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	members, _, err := h.orgStore.ListOrgMembers(ctx, org.ID(), port.ListOrgMembersOptions{})
	if err != nil {
		slog.WarnContext(ctx, "could not list org members for CSV", slogx.Error(err))
	}

	userNames := make(map[model.UserID]string)
	for _, m := range members {
		userNames[m.UserID()] = m.User().DisplayName()
	}

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

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-usage-%s.csv\"", org.Slug(), time.Now().Format("2006-01-02")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Utilisateur", "Modèle", "Tokens prompt", "Tokens completion", "Coût", "Devise", "Énergie (Wh)", "CO₂ (g)", "Date"})

	for _, rec := range records {
		userName := userNames[rec.UserID()]
		if userName == "" {
			userName = string(rec.UserID())
		}

		modelName := rec.ProxyModelName()
		if rec.ResolvedModelName() != "" && rec.ResolvedModelName() != rec.ProxyModelName() {
			modelName = rec.ProxyModelName() + " → " + rec.ResolvedModelName()
		}

		var energyWh, co2Grams float64
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
			energyWh = est.Mid.TotalWh
			co2Grams = est.Mid.Equivalences.CO2Grams
		}

		cost := float64(rec.Cost()) / 1_000_000

		writer.Write([]string{
			userName,
			modelName,
			strconv.Itoa(rec.PromptTokens()),
			strconv.Itoa(rec.CompletionTokens()),
			fmt.Sprintf("%.6f", cost),
			rec.Currency(),
			fmt.Sprintf("%.6f", energyWh),
			fmt.Sprintf("%.6f", co2Grams),
			rec.CreatedAt().Format("2006-01-02 15:04:05"),
		})
	}
}
