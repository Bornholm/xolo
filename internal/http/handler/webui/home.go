package webui

import (
	"net/http"

	httpCtx "github.com/bornholm/xolo/internal/http/context"
)

func (h *Handler) getHomePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := httpCtx.BaseURL(ctx)
	user := httpCtx.User(ctx)

	memberships := httpCtx.Memberships(ctx)

	if user == nil || len(memberships) == 0 {
		http.Redirect(w, r, baseURL.JoinPath("/no-org").String(), http.StatusTemporaryRedirect)
		return
	}

	http.Redirect(w, r, baseURL.JoinPath("/usage").String(), http.StatusTemporaryRedirect)
}
