package org

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/pkg/errors"
)

func (h *Handler) getOrgQuotaPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	quotaStore, ok := h.orgStore.(port.QuotaStore)
	if !ok {
		http.Error(w, "Quota store not available", http.StatusInternalServerError)
		return
	}

	existing, _ := quotaStore.GetQuota(ctx, model.QuotaScopeOrg, string(org.ID()))

	vmodel := component.QuotaPageVModel{
		Org:       org,
		ScopeType: "org",
		ScopeID:   string(org.ID()),
		Quota:     existing,
		Success:   r.URL.Query().Get("success"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "org-" + orgSlug + "-quota",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/admin/"},
				{Label: "Quotas", Href: "/orgs/" + orgSlug + "/admin/quota"},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.QuotaPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) saveOrgQuota(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	currency := org.Currency()
	if currency == "" {
		currency = model.DefaultCurrency
	}
	daily := parseBudgetField(r.FormValue("daily_budget"))
	monthly := parseBudgetField(r.FormValue("monthly_budget"))
	yearly := parseBudgetField(r.FormValue("yearly_budget"))

	quotaStore, ok := h.orgStore.(port.QuotaStore)
	if !ok {
		http.Error(w, "Quota store not available", http.StatusInternalServerError)
		return
	}

	quota := model.NewQuota(model.QuotaScopeOrg, string(org.ID()), currency, daily, monthly, yearly)
	if err := quotaStore.SetQuota(ctx, quota); err != nil {
		slog.ErrorContext(ctx, "could not save org quota", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/quota?success=saved", http.StatusSeeOther)
}

func (h *Handler) getMemberQuotaPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)
	membershipID := r.PathValue("membershipID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	membership, err := h.orgStore.GetMembership(ctx, model.MembershipID(membershipID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Membership not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	quotaStore, ok := h.orgStore.(port.QuotaStore)
	if !ok {
		http.Error(w, "Quota store not available", http.StatusInternalServerError)
		return
	}

	existing, _ := quotaStore.GetQuota(ctx, model.QuotaScopeUser, string(membership.UserID()))

	vmodel := component.QuotaPageVModel{
		Org:        org,
		Membership: membership,
		ScopeType:  "user",
		ScopeID:    string(membership.UserID()),
		Quota:      existing,
		Success:    r.URL.Query().Get("success"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "org-" + orgSlug + "-members",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/admin/"},
				{Label: "Membres", Href: "/orgs/" + orgSlug + "/admin/members"},
				{Label: membership.User().DisplayName(), Href: ""},
				{Label: "Quotas", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.QuotaPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) saveMemberQuota(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	membershipID := r.PathValue("membershipID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	membership, err := h.orgStore.GetMembership(ctx, model.MembershipID(membershipID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Membership not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	currency := org.Currency()
	if currency == "" {
		currency = model.DefaultCurrency
	}
	daily := parseBudgetField(r.FormValue("daily_budget"))
	monthly := parseBudgetField(r.FormValue("monthly_budget"))
	yearly := parseBudgetField(r.FormValue("yearly_budget"))

	quotaStore, ok := h.orgStore.(port.QuotaStore)
	if !ok {
		http.Error(w, "Quota store not available", http.StatusInternalServerError)
		return
	}

	quota := model.NewQuota(model.QuotaScopeUser, string(membership.UserID()), currency, daily, monthly, yearly)
	if err := quotaStore.SetQuota(ctx, quota); err != nil {
		slog.ErrorContext(ctx, "could not save member quota", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/members/"+membershipID+"/quota?success=saved", http.StatusSeeOther)
}

// parseBudgetField parses a currency budget field into microcents. Empty → nil (unlimited).
func parseBudgetField(v string) *int64 {
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return nil
	}
	mc := int64(f * 1_000_000)
	return &mc
}
