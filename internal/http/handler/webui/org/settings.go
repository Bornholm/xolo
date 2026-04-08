package org

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
)

func (h *Handler) getSettingsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	vmodel := component.OrgSettingsPageVModel{
		Org:     org,
		Success: r.URL.Query().Get("success"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "org-" + orgSlug + "-settings",
			HomeLink:      "/orgs/" + orgSlug,
			AdminSubtitle: "Admin. " + org.Name(),
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Paramètres", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	if org.ShareQuotaEqually() {
		orgQuota, err := h.quotaStore.GetQuota(ctx, model.QuotaScopeOrg, string(org.ID()))
		if err == nil && orgQuota != nil {
			members, _, err := h.orgStore.ListOrgMembers(ctx, org.ID(), port.ListOrgMembersOptions{})
			if err == nil && len(members) > 0 {
				n := int64(len(members))
				vmodel.MemberCount = len(members)
				if v := orgQuota.DailyBudget(); v != nil {
					divided := *v / n
					vmodel.SharedDailyBudget = &divided
				}
				if v := orgQuota.MonthlyBudget(); v != nil {
					divided := *v / n
					vmodel.SharedMonthlyBudget = &divided
				}
				if v := orgQuota.YearlyBudget(); v != nil {
					divided := *v / n
					vmodel.SharedYearlyBudget = &divided
				}
			}
		}
	}

	templ.Handler(component.OrgSettingsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) saveSettings(w http.ResponseWriter, r *http.Request) {
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

	currency := r.FormValue("currency")
	if currency == "" {
		currency = model.DefaultCurrency
	}

	shareQuotaEqually := r.FormValue("share_quota_equally") == "on"

	updated := model.UpdateOrganization(org,
		model.WithOrgCurrency(currency),
		model.WithOrgShareQuotaEqually(shareQuotaEqually),
	)
	if err := h.orgStore.SaveOrg(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "could not save org settings", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/settings?success=saved", http.StatusSeeOther)
}
