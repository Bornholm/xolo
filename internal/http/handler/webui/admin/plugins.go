package admin

import (
	"net/http"
	"slices"

	"github.com/a-h/templ"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/handler/webui/admin/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	"github.com/pkg/errors"
)

func (h *Handler) getPluginsDiagnosticsPage(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillPluginsDiagnosticsPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}
	templ.Handler(component.PluginsDiagnosticsPage(*vmodel)).ServeHTTP(w, r)
}

func (h *Handler) fillPluginsDiagnosticsPageViewModel(r *http.Request) (*component.PluginsDiagnosticsPageVModel, error) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	if user == nil {
		return nil, errors.New("could not retrieve user from context")
	}

	isAdmin := slices.Contains(user.Roles(), authz.RoleAdmin)

	vmodel := &component.PluginsDiagnosticsPageVModel{
		AppLayoutVModel: commonComp.AppLayoutVModel{
			User:          user,
			IsAdmin:       isAdmin,
			SelectedItem:  "plugins",
			AdminSubtitle: "Admin. plateforme",
			Breadcrumbs: []commonComp.BreadcrumbItem{
				{Label: "Plateforme", Href: "/admin/"},
				{Label: "Plugins", Href: ""},
			},
			NavigationItems: func(vmodel commonComp.AppLayoutVModel) templ.Component {
				return commonComp.AdminNavigationItems(vmodel)
			},
			FooterItems: func(vmodel commonComp.AppLayoutVModel) templ.Component {
				return commonComp.AdminFooterItems(vmodel)
			},
		},
		Descriptors: h.pluginManager.List(),
	}

	return vmodel, nil
}
