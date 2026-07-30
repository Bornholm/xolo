package admin

import (
	"net/http"

	"github.com/a-h/templ"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/handler/webui/admin/component"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

// getProxyHealthPage serves the proxy health screen.
//
// TODO(rework-ux): the screen has no data source. Reporting upstream health
// needs three things the domain does not have: a prober keeping the last known
// state of every provider endpoint, a rolling counter of response codes, and a
// latency histogram. The proxy records usage after the fact, not availability.
// The screen is served with its target structure so the information
// architecture of the mockup is in place (cf. lot 7 du plan).
func (h *Handler) getProxyHealthPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vmodel := component.ProxyHealthPageVModel{
		AppLayoutVModel: common.AppLayoutVModel{
			User:         httpCtx.User(ctx),
			IsAdmin:      true,
			SelectedItem: "health",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Plateforme", Href: "/admin/"},
				{Label: "Santé du proxy", Href: ""},
			},
			Context: common.ContextPlatform,
		},
	}

	templ.Handler(component.ProxyHealthPage(vmodel)).ServeHTTP(w, r)
}
