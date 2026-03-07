package oidc

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/xolo/internal/build"
	"github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	oidccomponent "github.com/bornholm/xolo/internal/http/middleware/authn/oidc/component"
)

func (h *Handler) getLoginPage(w http.ResponseWriter, r *http.Request) {
	vmodel := oidccomponent.LoginPageVModel{
		Providers: h.providers,
		Version:   build.ShortVersion,
		APIDocURL: string(component.BaseURL(r.Context(), component.WithPath("/docs/index.html"))),
	}

	loginPage := oidccomponent.LoginPage(vmodel)

	templ.Handler(loginPage).ServeHTTP(w, r)
}
