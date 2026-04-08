package org

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/pkg/errors"
)

func (h *Handler) getDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		slog.ErrorContext(ctx, "could not load org", slogx.Error(err))
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	members, _, err := h.orgStore.ListOrgMembers(ctx, org.ID(), port.ListOrgMembersOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "could not list members", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	providers, err := h.providerStore.ListProviders(ctx, org.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list providers", slogx.Error(errors.WithStack(err)))
		providers = nil
	}

	vmodel := component.OrgDashboardVModel{
		Org:       org,
		Members:   members,
		Providers: providers,
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "org-" + orgSlug,
			HomeLink:      "/orgs/" + orgSlug,
			AdminSubtitle: "Admin. " + org.Name(),
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Tableau de bord", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.OrgDashboard(vmodel)).ServeHTTP(w, r)
}
