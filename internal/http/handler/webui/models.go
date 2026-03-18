package webui

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/profile/component"
)

func (h *Handler) getModelsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	memberships := httpCtx.Memberships(ctx)

	rangeParam := r.URL.Query().Get("range")
	since := dashboardRangeToSince(rangeParam)

	userID := user.ID()
	var modelUsages []component.ModelUsage
	for _, m := range memberships {
		models, err := h.providerStore.ListEnabledLLMModels(ctx, m.OrgID())
		if err != nil {
			slog.ErrorContext(ctx, "could not list models", slogx.Error(err), slog.String("orgID", string(m.OrgID())))
			continue
		}
		for _, llmModel := range models {
			modelID := llmModel.ID()
			modelAgg, err := h.usageStore.AggregateUsage(ctx, port.UsageFilter{
				UserID:  &userID,
				ModelID: &modelID,
				Since:   &since,
			})
			if err != nil {
				slog.ErrorContext(ctx, "could not aggregate model usage", slogx.Error(err))
				modelAgg = nil
			}
			var org model.Organization
			if m.Org() != nil {
				org = m.Org()
			}
			modelUsages = append(modelUsages, component.ModelUsage{
				Model:     llmModel,
				Org:       org,
				Aggregate: modelAgg,
			})
		}
	}

	vmodel := component.ModelsPageVModel{
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "models",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace de travail", Href: "/usage"},
				{Label: "Modèles", Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
		ModelUsages: modelUsages,
		Range:       rangeParam,
	}

	templ.Handler(component.ModelsPage(vmodel)).ServeHTTP(w, r)
}
