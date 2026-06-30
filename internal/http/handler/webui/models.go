package webui

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/rbac"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/profile/component"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

func (h *Handler) getModelsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	memberships := httpCtx.Memberships(ctx)

	rangeParam := r.URL.Query().Get("range")
	since := dashboardRangeToSince(rangeParam)
	showAll := r.URL.Query().Get("show_all") == "true"

	modelUsages := h.loadModelUsages(ctx, user.ID(), memberships, since)

	sort.Slice(modelUsages, func(i, j int) bool {
		a := modelUsages[i].Aggregate
		b := modelUsages[j].Aggregate
		if a == nil && b == nil {
			return false
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.TotalRequests > b.TotalRequests
	})

	allModelUsages := modelUsages
	totalCount := len(modelUsages)
	remainingCount := totalCount - component.DefaultMaxDisplayedModels

	if !showAll && totalCount > component.DefaultMaxDisplayedModels {
		modelUsages = modelUsages[:component.DefaultMaxDisplayedModels]
	} else {
		remainingCount = 0
	}

	vmodel := component.ModelsPageVModel{
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "models",
			HomeLink:     "/usage",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace personnel", Href: "/usage"},
				{Label: "Modèles", Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
		ModelUsages:    modelUsages,
		AllModelUsages: allModelUsages,
		Range:          rangeParam,
		RemainingCount: remainingCount,
	}

	templ.Handler(component.ModelsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) loadModelUsages(ctx context.Context, userID model.UserID, memberships []model.Membership, since time.Time) []component.ModelUsage {
	user := httpCtx.User(ctx)
	isGlobalAdmin := user != nil && slices.Contains(user.Roles(), authz.RoleAdmin)

	var modelUsages []component.ModelUsage
	for _, m := range memberships {
		perms, err := h.roleStore.ResolveEffectivePermissions(ctx, userID, m.OrgID())
		if err != nil {
			slog.ErrorContext(ctx, "could not resolve permissions", slogx.Error(err), slog.String("orgID", string(m.OrgID())))
			continue
		}
		canUseOrg := isGlobalAdmin || perms.IsOwner() || perms.Has(rbac.PermModelUseOrg)
		canUseVirtual := isGlobalAdmin || perms.IsOwner() || perms.Has(rbac.PermModelUseVirtual)
		hasLLMGrants := perms.HasAnyGrant(rbac.ModelKindLLM)
		hasVirtualGrants := perms.HasAnyGrant(rbac.ModelKindVirtual)

		var org model.Organization
		if m.Org() != nil {
			org = m.Org()
		}

		// Load regular LLM models (blanket permission or per-model grants)
		if canUseOrg || hasLLMGrants {
			models, err := h.providerStore.ListEnabledLLMModels(ctx, m.OrgID())
			if err != nil {
				slog.ErrorContext(ctx, "could not list models", slogx.Error(err), slog.String("orgID", string(m.OrgID())))
			} else {
				providerCache := make(map[model.ProviderID]model.Provider)
				for _, llmModel := range models {
					if !canUseOrg && !perms.HasModelAccess(string(llmModel.ID()), rbac.ModelKindLLM) {
						continue
					}
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
					providerID := llmModel.ProviderID()
					if _, ok := providerCache[providerID]; !ok {
						p, err := h.providerStore.GetProviderByID(ctx, providerID)
						if err != nil {
							slog.ErrorContext(ctx, "could not get provider", slogx.Error(err), slog.String("providerID", string(providerID)))
						} else {
							providerCache[providerID] = p
						}
					}
					modelUsages = append(modelUsages, component.ModelUsage{
						Model:     llmModel,
						Org:       org,
						Aggregate: modelAgg,
						Provider:  providerCache[providerID],
					})
				}
			}
		}

		// Load virtual models (blanket permission or per-model grants)
		if h.virtualModelStore != nil && (canUseVirtual || hasVirtualGrants) {
			vms, err := h.virtualModelStore.ListVirtualModels(ctx, m.OrgID())
			if err != nil {
				slog.ErrorContext(ctx, "could not list virtual models", slogx.Error(err), slog.String("orgID", string(m.OrgID())))
			} else {
				for _, vm := range vms {
					if !canUseVirtual && !perms.HasModelAccess(string(vm.ID()), rbac.ModelKindVirtual) {
						continue
					}
					qualifiedName := org.Slug() + "/" + vm.Name()
					modelAgg, err := h.usageStore.AggregateUsage(ctx, port.UsageFilter{
						UserID:         &userID,
						ProxyModelName: &qualifiedName,
						Since:          &since,
					})
					if err != nil {
						slog.ErrorContext(ctx, "could not aggregate virtual model usage", slogx.Error(err))
						modelAgg = nil
					}
					wrappedModel := &virtualModelAsLLMModel{vm: vm, org: org}
					modelUsages = append(modelUsages, component.ModelUsage{
						Model:     wrappedModel,
						Org:       org,
						Aggregate: modelAgg,
					})
				}
			}
		}
	}
	return modelUsages
}
