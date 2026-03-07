package token

import (
	"log/slog"
	"net/http"

	"github.com/bornholm/go-x/slogx"
	"github.com/pkg/errors"
)

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.clearSession(w, r); err != nil && !errors.Is(err, errSessionNotFound) {
		slog.ErrorContext(ctx, "could not clear session", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
