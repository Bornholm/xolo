package webui

import (
	"net/http"
	"slices"

	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

func (h *Handler) getHomePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := httpCtx.BaseURL(ctx)
	user := httpCtx.User(ctx)

	// Platform admin → platform administration
	if user != nil && slices.Contains(user.Roles(), authz.RoleAdmin) {
		http.Redirect(w, r, baseURL.JoinPath("/admin/users").String(), http.StatusTemporaryRedirect)
		return
	}

	// Org admin (first admin org) → org administration
	if user != nil {
		memberships := httpCtx.Memberships(ctx)
		if org := firstAdminOrg(memberships); org != nil {
			if len(memberships) > 0 {
				http.Redirect(w, r, baseURL.JoinPath("/orgs/", org.Slug(), "/admin/").String(), http.StatusTemporaryRedirect)
				return
			}
		}

		// Regular user with memberships → usage
		if len(memberships) > 0 {
			http.Redirect(w, r, baseURL.JoinPath("/usage").String(), http.StatusTemporaryRedirect)
			return
		}

		// Regular user without memberships → no-org
		http.Redirect(w, r, baseURL.JoinPath("/no-org").String(), http.StatusTemporaryRedirect)
		return
	}

	// Fallback → usage
	http.Redirect(w, r, baseURL.JoinPath("/usage").String(), http.StatusTemporaryRedirect)
}

func firstAdminOrg(memberships []model.Membership) model.Organization {
	for _, m := range memberships {
		if m.Role() == model.RoleOrgAdmin || m.Role() == model.RoleOrgOwner {
			return m.Org()
		}
	}
	return nil
}
