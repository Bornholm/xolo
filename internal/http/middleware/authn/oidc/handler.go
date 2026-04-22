package oidc

import (
	"context"
	"net/http"

	"github.com/bornholm/xolo/internal/http/middleware/authn/oidctoken"
	"github.com/gorilla/sessions"
)

type ProviderWithJWKS struct {
	ID          string
	Label      string
	Icon       string
	DiscoveryURL string
	Issuer      string
	JWKSURL     string
}

type Handler struct {
	mux              *http.ServeMux
	sessionStore     sessions.Store
	sessionName      string
	providers       []Provider
	providersWithJWKS []ProviderWithJWKS
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(sessionStore sessions.Store, funcs ...OptionFunc) *Handler {
	opts := NewOptions(funcs...)
	h := &Handler{
		mux:              http.NewServeMux(),
		sessionStore:     sessionStore,
		sessionName:      opts.SessionName,
		providers:       opts.Providers,
		providersWithJWKS: opts.ProvidersWithJWKS,
	}

	h.mux.HandleFunc("GET /login", h.getLoginPage)
	h.mux.Handle("GET /providers/{provider}", withContextProvider(http.HandlerFunc(h.handleProvider)))
	h.mux.Handle("GET /providers/{provider}/callback", withContextProvider(http.HandlerFunc(h.handleProviderCallback)))
	h.mux.HandleFunc("GET /logout", h.handleLogout)
	h.mux.Handle("GET /providers/{provider}/logout", withContextProvider(http.HandlerFunc(h.handleProviderLogout)))

	return h
}

func (h *Handler) ProvidersWithJWKS() []oidctoken.Provider {
	providers := make([]oidctoken.Provider, 0, len(h.providersWithJWKS))
	for _, p := range h.providersWithJWKS {
		providers = append(providers, oidctoken.Provider{
			ID:          p.ID,
			Label:      p.Label,
			Icon:       p.Icon,
			DiscoveryURL: p.DiscoveryURL,
			Issuer:      p.Issuer,
			JWKSURL:     p.JWKSURL,
		})
	}
	return providers
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
