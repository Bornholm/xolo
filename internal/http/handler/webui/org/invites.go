package org

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/bornholm/go-x/slogx"
	"github.com/pkg/errors"
)

func (h *Handler) getInvitesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	invites, err := h.inviteStore.ListInvites(ctx, org.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list invites", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	baseURL := httpCtx.BaseURL(ctx)

	vmodel := component.InvitesPageVModel{
		Org:     org,
		Invites: invites,
		BaseURL: baseURL.String(),
		Success: r.URL.Query().Get("success"),
		NewURL:  r.URL.Query().Get("new_url"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-invites",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.InvitesPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getNewInvitePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	vmodel := component.InviteFormVModel{
		Org: org,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-invites",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.InviteForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) createInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	user := httpCtx.User(ctx)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	role := r.FormValue("role")
	if role == "" {
		role = model.RoleMember
	}

	var inviteeEmail *string
	if email := r.FormValue("invitee_email"); email != "" {
		inviteeEmail = &email
	}

	var expiresAt *time.Time
	if exp := r.FormValue("expires_at"); exp != "" {
		t, err := time.Parse("2006-01-02", exp)
		if err == nil {
			expiresAt = &t
		}
	}

	var maxUses *int
	if mu := r.FormValue("max_uses"); mu != "" {
		n, err := strconv.Atoi(mu)
		if err == nil && n > 0 {
			maxUses = &n
		}
	}

	invite := model.NewInviteToken(org.ID(), role, inviteeEmail, expiresAt, maxUses, user.ID())

	if err := h.inviteStore.CreateInvite(ctx, invite); err != nil {
		slog.ErrorContext(ctx, "could not create invite", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	baseURL := httpCtx.BaseURL(ctx)
	joinURL := baseURL.JoinPath("/join/" + string(invite.ID())).String()

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/invites?success=created&new_url="+joinURL, http.StatusSeeOther)
}

func (h *Handler) revokeInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	inviteID := r.PathValue("inviteID")

	if err := h.inviteStore.RevokeInvite(ctx, model.InviteTokenID(inviteID)); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Invite not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(ctx, "could not revoke invite", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/invites?success=revoked", http.StatusSeeOther)
}
