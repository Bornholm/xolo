package oidc

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
)

type Handler struct {
	mux          *http.ServeMux
	sessionStore sessions.Store
	sessionName  string
	providers    []Provider
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(sessionStore sessions.Store, funcs ...OptionFunc) *Handler {
	opts := NewOptions(funcs...)
	h := &Handler{
		mux:          http.NewServeMux(),
		sessionStore: sessionStore,
		sessionName:  opts.SessionName,
		providers:    opts.Providers,
	}

	h.mux.HandleFunc("GET /login", h.getLoginPage)
	h.mux.Handle("GET /providers/{provider}", withContextProvider(http.HandlerFunc(h.handleProvider)))
	h.mux.Handle("GET /providers/{provider}/callback", withContextProvider(http.HandlerFunc(h.handleProviderCallback)))
	h.mux.HandleFunc("GET /logout", h.handleLogout)
	h.mux.Handle("GET /providers/{provider}/logout", withContextProvider(http.HandlerFunc(h.handleProviderLogout)))

	return h
}

var _ http.Handler = &Handler{}

func withContextProvider(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		r = r.WithContext(context.WithValue(r.Context(), "provider", provider))
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
