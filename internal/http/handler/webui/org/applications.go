package org

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/pkg/errors"
)

func (h *Handler) getApplicationsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	user := httpCtx.User(ctx)
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(ctx, "could not get org", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	apps, err := h.applicationStore.QueryApplications(ctx, org.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list applications", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.ApplicationsPageVModel{
		Org:     org,
		Apps:    apps,
		Success: r.URL.Query().Get("success"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "org-" + orgSlug + "-applications",
			HomeLink:      "/orgs/" + orgSlug + "/admin/",
			AdminSubtitle: "Admin. " + org.Name(),
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/admin/"},
				{Label: "Applications", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ApplicationsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getNewApplicationPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	user := httpCtx.User(ctx)
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.ApplicationFormVModel{
		Org:   org,
		IsNew: true,
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "org-" + orgSlug + "-applications",
			HomeLink:      "/orgs/" + orgSlug + "/admin/",
			AdminSubtitle: "Admin. " + org.Name(),
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/admin/"},
				{Label: "Applications", Href: "/orgs/" + orgSlug + "/admin/applications"},
				{Label: "Nouvelle application", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ApplicationForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) createApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	app := model.NewApplication(org.ID(), name, description, true)
	if err := h.applicationStore.CreateApplication(ctx, app); err != nil {
		slog.ErrorContext(ctx, "could not create application", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/applications?success=created", http.StatusSeeOther)
}

func (h *Handler) getEditApplicationPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	appID := r.PathValue("appID")
	user := httpCtx.User(ctx)
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	app, err := h.applicationStore.GetApplication(ctx, model.ApplicationID(appID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Application not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tokens, err := h.applicationStore.GetApplicationAuthTokens(ctx, app.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list tokens", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.ApplicationFormVModel{
		Org:    org,
		App:    app,
		Tokens: tokens,
		IsNew:  false,
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "org-" + orgSlug + "-applications",
			HomeLink:      "/orgs/" + orgSlug + "/admin/",
			AdminSubtitle: "Admin. " + org.Name(),
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/admin/"},
				{Label: "Applications", Href: "/orgs/" + orgSlug + "/admin/applications"},
				{Label: app.Name(), Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ApplicationForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updateApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	appID := r.PathValue("appID")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	active := r.FormValue("active") == "on"

	app, err := h.applicationStore.GetApplication(ctx, model.ApplicationID(appID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Application not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	updated := model.UpdateApplication(app,
		model.WithApplicationName(name),
		model.WithApplicationDescription(description),
		model.WithApplicationActive(active),
	)

	if err := h.applicationStore.UpdateApplication(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "could not update application", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/applications?success=saved", http.StatusSeeOther)
}

func (h *Handler) deleteApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	appID := r.PathValue("appID")

	if err := h.applicationStore.DeleteApplication(ctx, model.ApplicationID(appID)); err != nil {
		slog.ErrorContext(ctx, "could not delete application", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/applications?success=deleted", http.StatusSeeOther)
}

func (h *Handler) createApplicationToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	appID := r.PathValue("appID")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	label := r.FormValue("label")
	expiryDays := r.FormValue("expiry")

	if label == "" {
		http.Error(w, "Label is required", http.StatusBadRequest)
		return
	}

	app, err := h.applicationStore.GetApplication(ctx, model.ApplicationID(appID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Application not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tokenValue := generateTokenValue()
	var expiresAt *time.Time
	if expiryDays != "" {
		days := 365
		if _, err := fmt.Sscanf(expiryDays, "%d", &days); err == nil {
			t := time.Now().Add(time.Duration(days) * 24 * time.Hour)
			expiresAt = &t
		}
	}

	token := model.NewApplicationAuthToken(app, org.ID(), label, tokenValue, expiresAt)
	if err := h.applicationStore.CreateApplicationAuthToken(ctx, token); err != nil {
		slog.ErrorContext(ctx, "could not create token", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/applications/"+appID+"/edit?success=token-created", http.StatusSeeOther)
}

func (h *Handler) deleteApplicationToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	appID := r.PathValue("appID")
	tokenID := r.PathValue("tokenID")

	if err := h.applicationStore.DeleteApplicationAuthToken(ctx, model.AuthTokenID(tokenID)); err != nil {
		slog.ErrorContext(ctx, "could not delete token", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/applications/"+appID+"/edit?success=token-deleted", http.StatusSeeOther)
}

func generateTokenValue() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "xapp_" + hex.EncodeToString(b)
}
