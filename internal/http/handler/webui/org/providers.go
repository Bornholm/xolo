package org

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/genai/llm/provider"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/crypto"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/bornholm/go-x/slogx"
	"github.com/pkg/errors"

	_ "github.com/bornholm/genai/llm/provider/openai"
	_ "github.com/bornholm/genai/llm/provider/mistral"
	_ "github.com/bornholm/genai/llm/provider/openrouter"
)

func (h *Handler) getProvidersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	providers, err := h.providerStore.ListProviders(ctx, org.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list providers", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.ProvidersPageVModel{
		Org:       org,
		Providers: providers,
		Success:   r.URL.Query().Get("success"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-providers",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ProvidersPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getNewProviderPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	vmodel := component.ProviderFormVModel{
		Org:   org,
		IsNew: true,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-providers",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ProviderForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) createProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	apiKey := r.FormValue("api_key")
	encryptedKey, err := crypto.Encrypt(h.secretKey, apiKey)
	if err != nil {
		slog.ErrorContext(ctx, "could not encrypt API key", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	p := model.NewProvider(org.ID(), r.FormValue("name"), r.FormValue("provider_type"), strings.TrimSpace(r.FormValue("base_url")), encryptedKey, r.FormValue("currency"))
	if err := h.providerStore.CreateProvider(ctx, p); err != nil {
		slog.ErrorContext(ctx, "could not create provider", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/providers?success=created", http.StatusSeeOther)
}

func (h *Handler) getEditProviderPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)
	providerID := r.PathValue("providerID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	p, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.ProviderFormVModel{
		Org:      org,
		Provider: p,
		IsNew:    false,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-providers",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ProviderForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updateProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	providerID := r.PathValue("providerID")

	_, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	existing, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	apiKey := existing.APIKey()
	if newKey := r.FormValue("api_key"); newKey != "" {
		apiKey, err = crypto.Encrypt(h.secretKey, newKey)
		if err != nil {
			slog.ErrorContext(ctx, "could not encrypt API key", slogx.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	currency := r.FormValue("currency")
	if currency == "" {
		currency = existing.Currency()
	}

	updated := &updatedProviderAdapter{
		id:        existing.ID(),
		orgID:     existing.OrgID(),
		name:      r.FormValue("name"),
		pType:     r.FormValue("provider_type"),
		baseURL:   strings.TrimSpace(r.FormValue("base_url")),
		apiKey:    apiKey,
		active:    r.FormValue("active") == "on",
		currency:  currency,
		createdAt: existing.CreatedAt(),
		updatedAt: time.Now(),
	}

	if err := h.providerStore.SaveProvider(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "could not save provider", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/providers?success=updated", http.StatusSeeOther)
}

func (h *Handler) deleteProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	providerID := r.PathValue("providerID")

	if err := h.providerStore.DeleteProvider(ctx, model.ProviderID(providerID)); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(ctx, "could not delete provider", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/providers?success=deleted", http.StatusSeeOther)
}

func (h *Handler) testProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	providerID := r.PathValue("providerID")

	p, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<span class="text-destructive">Provider not found</span>`))
		return
	}

	decryptedKey, err := crypto.Decrypt(h.secretKey, p.APIKey())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<span class="text-destructive">Could not decrypt API key</span>`))
		return
	}

	_, err = testProviderConnection(ctx, p.Type(), p.BaseURL(), decryptedKey)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<span class="text-destructive">Connection failed: ` + err.Error() + `</span>`))
		return
	}

	w.Write([]byte(`<span class="text-green-600">Connection successful ✓</span>`))
}

func testProviderConnection(ctx context.Context, providerType, baseURL, apiKey string) (bool, error) {
	client, err := provider.Create(ctx,
		provider.WithChatCompletionOptions(provider.ClientOptions{
			Provider: provider.Name(providerType),
			BaseURL:  baseURL,
			APIKey:   apiKey,
			Model:    "gpt-3.5-turbo", // dummy model name — just validates credentials
		}),
	)
	if err != nil {
		return false, errors.WithStack(err)
	}
	_ = client
	return true, nil
}

func (h *Handler) getModelsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)
	providerID := r.PathValue("providerID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	p, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	models, err := h.providerStore.ListLLMModels(ctx, org.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list models", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Filter to this provider
	var filtered []model.LLMModel
	for _, m := range models {
		if m.ProviderID() == p.ID() {
			filtered = append(filtered, m)
		}
	}

	vmodel := component.ModelsPageVModel{
		Org:      org,
		Provider: p,
		Models:   filtered,
		Success:  r.URL.Query().Get("success"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-providers",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ModelsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getNewModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)
	providerID := r.PathValue("providerID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	p, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	vmodel := component.ModelFormVModel{
		Org:      org,
		Provider: p,
		IsNew:    true,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-providers",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ModelForm(vmodel)).ServeHTTP(w, r)
}

func parseCostField(v string) int64 {
	// Parse a dollar value per 1M tokens and convert to microcents per 1K tokens.
	// e.g. "2.50" ($/1M) → 2500 microcents/1K  (since 1M = 1000×1K, and 1$ = 1_000_000 microcents)
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return int64(f * 1_000)
}

func (h *Handler) createModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	providerID := r.PathValue("providerID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	p, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	proxyName := r.FormValue("proxy_name")

	// Reject duplicate proxy names within this organization.
	if conflicting, err := h.providerStore.GetLLMModelByProxyName(ctx, org.ID(), proxyName); err == nil && conflicting != nil {
		h.renderModelFormError(w, r, ctx, user, orgSlug, org, p, nil, true,
			"Un modèle avec le nom proxy « "+proxyName+" » existe déjà dans cette organisation.")
		return
	}

	m := model.NewLLMModel(
		p.ID(), org.ID(),
		proxyName,
		r.FormValue("real_model"),
		r.FormValue("description"),
		parseCostField(r.FormValue("prompt_cost")),
		parseCostField(r.FormValue("completion_cost")),
	)

	if err := h.providerStore.CreateLLMModel(ctx, m); err != nil {
		slog.ErrorContext(ctx, "could not create model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/providers/"+string(p.ID())+"/models?success=created", http.StatusSeeOther)
}

func (h *Handler) getEditModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)
	providerID := r.PathValue("providerID")
	modelID := r.PathValue("modelID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	p, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	m, err := h.providerStore.GetLLMModelByID(ctx, model.LLMModelID(modelID))
	if err != nil {
		http.Error(w, "Model not found", http.StatusNotFound)
		return
	}

	vmodel := component.ModelFormVModel{
		Org:      org,
		Provider: p,
		Model:    m,
		IsNew:    false,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-providers",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ModelForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updateModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	providerID := r.PathValue("providerID")
	modelID := r.PathValue("modelID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	p, err := h.providerStore.GetProviderByID(ctx, model.ProviderID(providerID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	existing, err := h.providerStore.GetLLMModelByID(ctx, model.LLMModelID(modelID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	proxyName := r.FormValue("proxy_name")

	// Reject duplicate proxy names within this organization (excluding the current model).
	if conflicting, err := h.providerStore.GetLLMModelByProxyName(ctx, org.ID(), proxyName); err == nil && conflicting != nil && conflicting.ID() != existing.ID() {
		h.renderModelFormError(w, r, ctx, user, orgSlug, org, p, existing, false,
			"Un modèle avec le nom proxy « "+proxyName+" » existe déjà dans cette organisation.")
		return
	}

	updated := &updatedLLMModelAdapter{
		id:                        existing.ID(),
		providerID:                existing.ProviderID(),
		orgID:                     existing.OrgID(),
		proxyName:                 proxyName,
		realModel:                 r.FormValue("real_model"),
		description:               r.FormValue("description"),
		enabled:                   r.FormValue("enabled") == "on",
		promptCostPer1KTokens:     parseCostField(r.FormValue("prompt_cost")),
		completionCostPer1KTokens: parseCostField(r.FormValue("completion_cost")),
		createdAt:                 existing.CreatedAt(),
		updatedAt:                 time.Now(),
	}

	if err := h.providerStore.SaveLLMModel(ctx, updated); err != nil {
		slog.ErrorContext(ctx, "could not save model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/providers/"+providerID+"/models?success=updated", http.StatusSeeOther)
}

func (h *Handler) renderModelFormError(w http.ResponseWriter, r *http.Request, ctx context.Context, user model.User, orgSlug string, org model.Organization, p model.Provider, m model.LLMModel, isNew bool, errMsg string) {
	nav, footer := orgAdminNav(orgSlug)
	vmodel := component.ModelFormVModel{
		Org:      org,
		Provider: p,
		Model:    m,
		IsNew:    isNew,
		Error:    errMsg,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-providers",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	templ.Handler(component.ModelForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) deleteModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	providerID := r.PathValue("providerID")
	modelID := r.PathValue("modelID")

	if err := h.providerStore.DeleteLLMModel(ctx, model.LLMModelID(modelID)); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(ctx, "could not delete model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/providers/"+providerID+"/models?success=deleted", http.StatusSeeOther)
}
