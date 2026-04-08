package admin

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/handler/webui/admin/component"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/pkg/errors"
)

func (h *Handler) getOrgsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	orgs, _, err := h.orgStore.ListOrgs(ctx, port.ListOrgsOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "could not list orgs", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.OrgsPageVModel{
		Orgs:    orgs,
		Success: r.URL.Query().Get("success"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "orgs",
			AdminSubtitle: "Admin. plateforme",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Plateforme", Href: "/admin/"},
				{Label: "Organisations", Href: "/admin/orgs"},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AdminNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AdminFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.OrgsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getNewOrgPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	vmodel := component.OrgFormVModel{
		IsNew: true,
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "orgs",
			AdminSubtitle: "Admin. plateforme",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Plateforme", Href: "/admin/"},
				{Label: "Organisations", Href: "/admin/orgs"},
				{Label: "Nouvelle organisation", Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AdminNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AdminFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.OrgForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) createOrg(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	slug := r.FormValue("slug")
	description := r.FormValue("description")

	if name == "" || slug == "" {
		http.Error(w, "Name and slug are required", http.StatusBadRequest)
		return
	}

	org := model.NewOrganization(slug, name, description)
	if err := h.orgStore.CreateOrg(ctx, org); err != nil {
		slog.ErrorContext(ctx, "could not create org", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/orgs?success=created", http.StatusSeeOther)
}

func (h *Handler) getEditOrgPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgID := r.PathValue("orgID")

	org, err := h.orgStore.GetOrgByID(ctx, model.OrgID(orgID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.OrgFormVModel{
		Org:   org,
		IsNew: false,
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "orgs",
			AdminSubtitle: "Admin. plateforme",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Plateforme", Href: "/admin/"},
				{Label: "Organisations", Href: "/admin/orgs"},
				{Label: org.Name(), Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AdminNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AdminFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.OrgForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updateOrg(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("orgID")

	org, err := h.orgStore.GetOrgByID(ctx, model.OrgID(orgID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	updated := model.UpdateOrganization(org,
		model.WithOrgName(r.FormValue("name")),
		model.WithOrgDescription(r.FormValue("description")),
		model.WithOrgActive(r.FormValue("active") == "on"),
	)

	if err := h.orgStore.SaveOrg(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "could not save org", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/orgs?success=saved", http.StatusSeeOther)
}
