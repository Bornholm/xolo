package admin

import (
	"context"
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

func (h *Handler) getExchangeRatesPage(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillExchangeRatesPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	page := component.ExchangeRatesPage(*vmodel)
	templ.Handler(page).ServeHTTP(w, r)
}

func (h *Handler) fillExchangeRatesPageViewModel(r *http.Request) (*component.ExchangeRatesPageVModel, error) {
	vmodel := &component.ExchangeRatesPageVModel{}
	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillExchangeRatesPageVModelAppLayout,
		h.fillExchangeRatesPageVModelRates,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillExchangeRatesPageVModelAppLayout(ctx context.Context, vmodel *component.ExchangeRatesPageVModel, r *http.Request) error {
	user := httpCtx.User(ctx)
	if user == nil {
		return errors.New("could not retrieve user from context")
	}

	isAdmin := slices.Contains(user.Roles(), authz.RoleAdmin)

	vmodel.AppLayoutVModel = commonComp.AppLayoutVModel{
		User:          user,
		IsAdmin:       isAdmin,
		SelectedItem:  "exchange-rates",
		HomeLink:      "/admin/",
		AdminSubtitle: "Admin. plateforme",
		Breadcrumbs: []commonComp.BreadcrumbItem{
			{Label: "Plateforme", Href: "/admin/"},
			{Label: "Taux de change", Href: ""},
		},
		NavigationItems: func(vmodel commonComp.AppLayoutVModel) templ.Component {
			return commonComp.AdminNavigationItems(vmodel)
		},
		FooterItems: func(vmodel commonComp.AppLayoutVModel) templ.Component {
			return commonComp.AdminFooterItems(vmodel)
		},
	}

	return nil
}

func (h *Handler) fillExchangeRatesPageVModelRates(ctx context.Context, vmodel *component.ExchangeRatesPageVModel, r *http.Request) error {
	rates, err := h.exchangeRateService.ListRates(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	vmodel.Rates = rates

	return nil
}
