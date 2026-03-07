package token

import (
	"net/http"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/gorilla/sessions"
)

type Handler struct {
	mux          *http.ServeMux
	sessionStore sessions.Store
	sessionName  string
	userStore    port.UserStore
}

// ServeHTTP implements [http.Handler].
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(sessionStore sessions.Store, userStore port.UserStore, funcs ...OptionFunc) *Handler {
	opts := NewOptions(funcs...)
	h := &Handler{
		mux:          http.NewServeMux(),
		sessionStore: sessionStore,
		sessionName:  opts.SessionName,
		userStore:    userStore,
	}

	h.mux.HandleFunc("GET /login", h.getLoginPage)
	h.mux.HandleFunc("POST /login", h.handleLogin)
	h.mux.HandleFunc("POST /logout", h.handleLogout)

	return h
}

var _ http.Handler = &Handler{}
