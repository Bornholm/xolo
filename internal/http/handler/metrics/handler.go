package metrics

import (
	"net/http"

	"github.com/bornholm/xolo/internal/http/middleware/authz"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	mux *http.ServeMux
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler() *Handler {
	h := &Handler{
		mux: &http.ServeMux{},
	}

	assertAuthenticated := authz.Middleware(nil, authz.IsAuthenticated)

	h.mux.Handle("GET /", assertAuthenticated(promhttp.Handler()))

	return h
}

var _ http.Handler = &Handler{}
