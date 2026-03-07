package admin

import (
	"net/http"

	httpCtx "github.com/bornholm/xolo/internal/http/context"
)

func (h *Handler) getIndexPage(w http.ResponseWriter, r *http.Request) {
	baseURL := httpCtx.BaseURL(r.Context())
	redirectURL := baseURL.JoinPath("/admin/users")
	http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
}
