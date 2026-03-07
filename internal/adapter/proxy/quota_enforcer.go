package proxy

import (
	"context"
	"fmt"
	"strings"
	"time"

	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

// XoloQuotaEnforcer is a PreRequestHook that checks the effective budget quota
// for the requesting user and org, rejecting requests that would exceed it.
type XoloQuotaEnforcer struct {
	quotaStore port.QuotaStore
	usageStore port.UsageStore
	userStore  port.UserStore
}

func NewXoloQuotaEnforcer(quotaStore port.QuotaStore, usageStore port.UsageStore, userStore port.UserStore) *XoloQuotaEnforcer {
	return &XoloQuotaEnforcer{
		quotaStore: quotaStore,
		usageStore: usageStore,
		userStore:  userStore,
	}
}

// populateMetaFromHeader looks up the Bearer token from the request Authorization
// header and stores orgID and authTokenID in req.Metadata.
//
// This is needed because the genai proxy captures ctx = r.Context() before calling
// the AuthExtractor, which modifies *r. Pre-request hooks therefore receive a stale
// ctx that does not carry the orgID set by XoloAuthExtractor. Reading the token
// directly from the Authorization header is the only reliable way to obtain orgID
// in the pre-request phase.
func (e *XoloQuotaEnforcer) populateMetaFromHeader(ctx context.Context, req *genaiProxy.ProxyRequest) {
	if OrgIDFromMeta(req.Metadata) != "" {
		return // already populated by a previous hook
	}
	auth := req.Headers.Get("Authorization")
	if auth == "" {
		return
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return
	}
	raw := strings.TrimSpace(parts[1])
	if raw == "" {
		return
	}
	token, err := e.userStore.FindAuthToken(ctx, raw)
	if err != nil {
		return
	}
	req.Metadata[MetaOrgID] = string(token.OrgID())
	req.Metadata[MetaAuthTokenID] = string(token.ID())
}

func (e *XoloQuotaEnforcer) Name() string  { return "xolo.quota-enforcer" }
func (e *XoloQuotaEnforcer) Priority() int { return 5 }

// PreRequest implements proxy.PreRequestHook.
func (e *XoloQuotaEnforcer) PreRequest(ctx context.Context, req *genaiProxy.ProxyRequest) (*genaiProxy.HookResult, error) {
	// The proxy captures ctx before running the AuthExtractor, so ctx is stale and
	// does not carry orgID. Read it directly from the Authorization header instead.
	e.populateMetaFromHeader(ctx, req)

	userID := model.UserID(req.UserID)
	orgID := OrgIDFromMeta(req.Metadata)

	if userID == "" || orgID == "" {
		// No auth context — let the request through (will fail at auth extractor level)
		return nil, nil
	}

	// ── Per-user quota check (effective = min of user quota and org quota) ──────
	effectiveQuota, err := e.quotaStore.ResolveEffectiveQuota(ctx, userID, orgID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	now := time.Now()
	currency := effectiveQuota.Currency

	if effectiveQuota.DailyBudget != nil {
		spent, err := e.usageStore.SumCostSince(ctx, userID, orgID, startOfDay(now))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if spent >= *effectiveQuota.DailyBudget {
			return &genaiProxy.HookResult{
				Response: rateLimitResponse(fmt.Sprintf(
					"Daily budget exceeded: %s / %s",
					formatMicrocents(spent, currency), formatMicrocents(*effectiveQuota.DailyBudget, currency),
				)),
			}, nil
		}
	}

	if effectiveQuota.MonthlyBudget != nil {
		spent, err := e.usageStore.SumCostSince(ctx, userID, orgID, startOfMonth(now))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if spent >= *effectiveQuota.MonthlyBudget {
			return &genaiProxy.HookResult{
				Response: rateLimitResponse(fmt.Sprintf(
					"Monthly budget exceeded: %s / %s",
					formatMicrocents(spent, currency), formatMicrocents(*effectiveQuota.MonthlyBudget, currency),
				)),
			}, nil
		}
	}

	if effectiveQuota.YearlyBudget != nil {
		spent, err := e.usageStore.SumCostSince(ctx, userID, orgID, startOfYear(now))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if spent >= *effectiveQuota.YearlyBudget {
			return &genaiProxy.HookResult{
				Response: rateLimitResponse(fmt.Sprintf(
					"Yearly budget exceeded: %s / %s",
					formatMicrocents(spent, currency), formatMicrocents(*effectiveQuota.YearlyBudget, currency),
				)),
			}, nil
		}
	}

	// ── Org-wide quota check (total spending by all users in the org) ──────────
	orgQuota, err := e.quotaStore.GetQuota(ctx, model.QuotaScopeOrg, string(orgID))
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, errors.WithStack(err)
	}
	if orgQuota != nil {
		orgCurrency := orgQuota.Currency()

		if orgQuota.DailyBudget() != nil {
			orgSpent, err := e.sumOrgCost(ctx, orgID, startOfDay(now))
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if orgSpent >= *orgQuota.DailyBudget() {
				return &genaiProxy.HookResult{
					Response: rateLimitResponse(fmt.Sprintf(
						"Organization daily budget exceeded: %s / %s",
						formatMicrocents(orgSpent, orgCurrency), formatMicrocents(*orgQuota.DailyBudget(), orgCurrency),
					)),
				}, nil
			}
		}

		if orgQuota.MonthlyBudget() != nil {
			orgSpent, err := e.sumOrgCost(ctx, orgID, startOfMonth(now))
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if orgSpent >= *orgQuota.MonthlyBudget() {
				return &genaiProxy.HookResult{
					Response: rateLimitResponse(fmt.Sprintf(
						"Organization monthly budget exceeded: %s / %s",
						formatMicrocents(orgSpent, orgCurrency), formatMicrocents(*orgQuota.MonthlyBudget(), orgCurrency),
					)),
				}, nil
			}
		}

		if orgQuota.YearlyBudget() != nil {
			orgSpent, err := e.sumOrgCost(ctx, orgID, startOfYear(now))
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if orgSpent >= *orgQuota.YearlyBudget() {
				return &genaiProxy.HookResult{
					Response: rateLimitResponse(fmt.Sprintf(
						"Organization yearly budget exceeded: %s / %s",
						formatMicrocents(orgSpent, orgCurrency), formatMicrocents(*orgQuota.YearlyBudget(), orgCurrency),
					)),
				}, nil
			}
		}
	}

	return nil, nil
}

// sumOrgCost returns the total cost for all users in the org since the given time,
// summing across all stored currencies. Because records are converted to org currency
// at record time, this approximates the true total in org currency.
func (e *XoloQuotaEnforcer) sumOrgCost(ctx context.Context, orgID model.OrgID, since time.Time) (int64, error) {
	byCurrency, err := e.usageStore.SumCostSinceByCurrency(ctx, nil, orgID, since)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	var total int64
	for _, amount := range byCurrency {
		total += amount
	}
	return total, nil
}

func rateLimitResponse(message string) *genaiProxy.ProxyResponse {
	return &genaiProxy.ProxyResponse{
		StatusCode: 429,
		Body: map[string]any{
			"error": map[string]any{
				"message": message,
				"type":    "rate_limit_error",
				"code":    "quota_exceeded",
			},
		},
	}
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func startOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func startOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// formatMicrocents converts microcents to a currency string, e.g. 1000000 USD → "$1.00".
func formatMicrocents(v int64, currency string) string {
	symbols := map[string]string{
		"EUR": "€", "GBP": "£", "JPY": "¥", "CHF": "CHF ", "CAD": "CA$", "AUD": "A$",
	}
	symbol := "$"
	if s, ok := symbols[currency]; ok {
		symbol = s
	}
	return fmt.Sprintf("%s%.2f", symbol, float64(v)/1_000_000)
}

var _ genaiProxy.PreRequestHook = &XoloQuotaEnforcer{}
