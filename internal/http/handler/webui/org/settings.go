package org

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/core/model"
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
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-settings",
			NavigationItems: nav,
			FooterItems:     footer,
		},
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

	updated := model.UpdateOrganization(org, model.WithOrgCurrency(currency))
	if err := h.orgStore.SaveOrg(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "could not save org settings", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/settings?success=saved", http.StatusSeeOther)
}
