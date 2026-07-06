package webui

import (
	"net/http"

	httpCtx "github.com/bornholm/xolo/internal/http/context"
)

// getPersonalEventsRedirect is the personal-menu entry point for events. It
// resolves the user's organization and redirects to that org's events explorer
// (which defaults to the user's own events). Users without an org land on the
// no-org page.
func (h *Handler) getPersonalEventsRedirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := httpCtx.BaseURL(ctx)

	memberships := httpCtx.Memberships(ctx)
	if len(memberships) == 0 {
		http.Redirect(w, r, baseURL.JoinPath("/no-org").String(), http.StatusTemporaryRedirect)
		return
	}

	org, err := h.orgStore.GetOrgByID(ctx, memberships[0].OrgID())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, baseURL.JoinPath("/orgs/"+org.Slug()+"/events").String()+"?view=personal", http.StatusSeeOther)
}
