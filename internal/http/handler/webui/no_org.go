package webui

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/profile/component"
)

func (h *Handler) getNoOrgPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	memberships := httpCtx.Memberships(ctx)

	baseURL := httpCtx.BaseURL(ctx)

	// If user has any membership, redirect to /usage
	if len(memberships) > 0 {
		http.Redirect(w, r, baseURL.JoinPath("/usage").String(), http.StatusTemporaryRedirect)
		return
	}

	// Fetch pending invitations for the user's email
	invites, err := h.inviteStore.ListPendingInvitesForEmail(ctx, user.Email())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch invitations", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Collect declined invite IDs from cookies
	var declinedIDs []string
	for _, inv := range invites {
		cookieName := fmt.Sprintf("declined_invite_%s", string(inv.ID()))
		if _, err := r.Cookie(cookieName); err == nil {
			declinedIDs = append(declinedIDs, string(inv.ID()))
		}
	}

	vmodel := component.NoOrgPageVModel{
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "usage",
			HomeLink:     "/usage",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace personnel", Href: "/usage"},
				{Label: "Aucune organisation", Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
		Invites:     invites,
		DeclinedIDs: declinedIDs,
	}

	templ.Handler(component.NoOrgPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) declineInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := httpCtx.BaseURL(ctx)

	tokenID := r.PathValue("tokenID")
	if tokenID == "" {
		http.Error(w, "Token ID is required", http.StatusBadRequest)
		return
	}

	invite, err := h.inviteStore.GetInviteByID(ctx, model.InviteTokenID(tokenID))
	if err != nil {
		// Invite not found or already gone — redirect silently.
		http.Redirect(w, r, baseURL.JoinPath("/no-org").String(), http.StatusSeeOther)
		return
	}

	// Targeted invites are deleted when declined; open invites just get a cookie.
	if invite.InviteeEmail() != nil {
		if err := h.inviteStore.DeleteInvite(ctx, invite.ID()); err != nil {
			slog.WarnContext(ctx, "could not delete targeted invite after decline", slogx.Error(err))
		}
	} else {
		cookieName := fmt.Sprintf("declined_invite_%s", tokenID)
		http.SetCookie(w, &http.Cookie{
			Name:   cookieName,
			Value:  "1",
			Path:   "/",
			MaxAge: 86400,
		})
	}

	http.Redirect(w, r, baseURL.JoinPath("/no-org").String(), http.StatusSeeOther)
}
