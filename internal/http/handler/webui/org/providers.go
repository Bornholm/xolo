package org

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/genai/llm/provider"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/crypto"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/pkg/errors"

	_ "github.com/bornholm/genai/llm/provider/mistral"
	_ "github.com/bornholm/genai/llm/provider/openai"
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
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: ""},
			},
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
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: "/orgs/" + orgSlug + "/admin/providers"},
				{Label: "Nouveau fournisseur", Href: ""},
			},
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

	cloudTier, _ := strconv.Atoi(r.FormValue("cloud_tier"))
	p := model.NewProvider(org.ID(), r.FormValue("name"), r.FormValue("provider_type"), strings.TrimSpace(r.FormValue("base_url")), encryptedKey, r.FormValue("currency"))
	p.SetCloudTier(cloudTier)
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
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: "/orgs/" + orgSlug + "/admin/providers"},
				{Label: p.Name(), Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ProviderForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updateProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	providerID := r.PathValue("providerID")

	org, err := h.orgFromSlug(ctx, orgSlug)
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

	// --- Retry config ---
	var retryConfig *model.RetryConfig
	if r.FormValue("retry_enabled") == "on" {
		delay, err := parseDurationField(r, "retry_delay_value", "retry_delay_unit")
		if err != nil || delay <= 0 {
			h.renderProviderFormError(w, r, ctx, user, orgSlug, org, existing, false,
				"Retry : le délai doit être un entier strictement positif.")
			return
		}
		attempts, _ := strconv.Atoi(r.FormValue("retry_max_attempts"))
		if attempts < 1 {
			h.renderProviderFormError(w, r, ctx, user, orgSlug, org, existing, false,
				"Retry : le nombre de tentatives doit être ≥ 1.")
			return
		}
		retryConfig = &model.RetryConfig{
			Enabled:     true,
			MaxAttempts: attempts,
			Delay:       delay,
		}
	}

	// --- Rate limit config ---
	var rateLimitConfig *model.RateLimitConfig
	if r.FormValue("rate_limit_enabled") == "on" {
		interval, err := parseDurationField(r, "rate_limit_interval_value", "rate_limit_interval_unit")
		if err != nil || interval <= 0 {
			h.renderProviderFormError(w, r, ctx, user, orgSlug, org, existing, false,
				"Rate limit : l'intervalle doit être un entier strictement positif.")
			return
		}
		burst, _ := strconv.Atoi(r.FormValue("rate_limit_max_burst"))
		if burst < 1 {
			h.renderProviderFormError(w, r, ctx, user, orgSlug, org, existing, false,
				"Rate limit : la capacité de burst doit être ≥ 1.")
			return
		}
		rateLimitConfig = &model.RateLimitConfig{
			Enabled:  true,
			Interval: interval,
			MaxBurst: burst,
		}
	}

	cloudTier, _ := strconv.Atoi(r.FormValue("cloud_tier"))

	updated := &updatedProviderAdapter{
		id:              existing.ID(),
		orgID:           existing.OrgID(),
		name:            r.FormValue("name"),
		pType:           r.FormValue("provider_type"),
		baseURL:         strings.TrimSpace(r.FormValue("base_url")),
		apiKey:          apiKey,
		active:          r.FormValue("active") == "on",
		currency:        currency,
		cloudTier:       cloudTier,
		createdAt:       existing.CreatedAt(),
		updatedAt:       time.Now(),
		retryConfig:     retryConfig,
		rateLimitConfig: rateLimitConfig,
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
	name := provider.Name(providerType)
	opts := provider.NewChatCompletionProviderOptions(name)
	if opts == nil {
		return false, errors.Errorf("unknown provider %q", providerType)
	}
	v := reflect.ValueOf(opts).Elem()
	if common := v.FieldByName("CommonOptions"); common.IsValid() {
		common.FieldByName("BaseURL").SetString(baseURL)
		common.FieldByName("APIKey").SetString(apiKey)
		// dummy model — just validates credentials
		common.FieldByName("Model").SetString("gpt-3.5-turbo")
	}
	client, err := provider.Create(ctx, func(o *provider.Options) error {
		o.ChatCompletion = &provider.ResolvedClientOptions{
			Provider: name,
			Specific: opts,
		}
		return nil
	})
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
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: "/orgs/" + orgSlug + "/admin/providers"},
				{Label: p.Name(), Href: "/orgs/" + orgSlug + "/admin/providers/" + string(p.ID()) + "/models"},
			},
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
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: "/orgs/" + orgSlug + "/admin/providers"},
				{Label: p.Name(), Href: "/orgs/" + orgSlug + "/admin/providers/" + string(p.ID()) + "/models"},
				{Label: p.Name(), Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.ModelForm(vmodel)).ServeHTTP(w, r)
}

func parseIntField(v string) int64 {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func parseFloat64Field(v string) float64 {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return 0
	}
	return f
}

// parseActiveParamsField parses a billions value (e.g. "7" for 7B) into raw int64.
// Returns 0 if the input is empty or invalid.
func parseActiveParamsField(v string) int64 {
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return int64(f * 1e9)
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

// parseDurationField reads a (value, unit) pair from the form and returns a time.Duration.
// value must be a positive integer; unit must be "ms", "s", or "min" (default: "s").
// Returns 0, nil if the value field is empty or absent.
// Returns an error if the value field is present but invalid or ≤ 0.
func parseDurationField(r *http.Request, valueField, unitField string) (time.Duration, error) {
	valueStr := r.FormValue(valueField)
	if valueStr == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil || v <= 0 {
		return 0, errors.Errorf("le champ %q doit être un entier strictement positif", valueField)
	}
	var multiplier time.Duration
	switch r.FormValue(unitField) {
	case "ms":
		multiplier = time.Millisecond
	case "min":
		multiplier = time.Minute
	default: // "s" or empty
		multiplier = time.Second
	}
	return time.Duration(v) * multiplier, nil
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
	m.SetContextWindow(parseIntField(r.FormValue("context_window")))
	m.SetOutputWindow(parseIntField(r.FormValue("output_window")))
	m.SetActiveParams(parseActiveParamsField(r.FormValue("active_params")))
	m.SetTokensPerSecLow(parseFloat64Field(r.FormValue("tokens_per_sec_low")))
	m.SetTokensPerSecHigh(parseFloat64Field(r.FormValue("tokens_per_sec_high")))
	m.SetCapabilities(model.ModelCapabilities{
		Tools:      r.FormValue("cap_tools") == "on",
		Vision:     r.FormValue("cap_vision") == "on",
		Reasoning:  r.FormValue("cap_reasoning") == "on",
		Audio:      r.FormValue("cap_audio") == "on",
		Embeddings: r.FormValue("cap_embeddings") == "on",
	})

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
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: "/orgs/" + orgSlug + "/admin/providers"},
				{Label: p.Name(), Href: "/orgs/" + orgSlug + "/admin/providers/" + string(p.ID()) + "/models"},
				{Label: m.ProxyName(), Href: ""},
			},
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

	// --- Token limit config ---
	var tokenLimitConfig *model.TokenLimitConfig
	if r.FormValue("token_limit_enabled") == "on" {
		interval, err := parseDurationField(r, "token_limit_interval_value", "token_limit_interval_unit")
		if err != nil || interval <= 0 {
			h.renderModelFormError(w, r, ctx, user, orgSlug, org, p, existing, false,
				"Limite de tokens : l'intervalle doit être un entier strictement positif.")
			return
		}
		maxTokens, _ := strconv.Atoi(r.FormValue("token_limit_max_tokens"))
		if maxTokens < 1 {
			h.renderModelFormError(w, r, ctx, user, orgSlug, org, p, existing, false,
				"Limite de tokens : le nombre de tokens doit être ≥ 1.")
			return
		}
		tokenLimitConfig = &model.TokenLimitConfig{
			Enabled:   true,
			MaxTokens: maxTokens,
			Interval:  interval,
		}
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
		contextWindow:             parseIntField(r.FormValue("context_window")),
		outputWindow:              parseIntField(r.FormValue("output_window")),
		activeParams:              parseActiveParamsField(r.FormValue("active_params")),
		tokensPerSecLow:           parseFloat64Field(r.FormValue("tokens_per_sec_low")),
		tokensPerSecHigh:          parseFloat64Field(r.FormValue("tokens_per_sec_high")),
		capabilities: model.ModelCapabilities{
			Tools:      r.FormValue("cap_tools") == "on",
			Vision:     r.FormValue("cap_vision") == "on",
			Reasoning:  r.FormValue("cap_reasoning") == "on",
			Audio:      r.FormValue("cap_audio") == "on",
			Embeddings: r.FormValue("cap_embeddings") == "on",
		},
		createdAt:        existing.CreatedAt(),
		updatedAt:        time.Now(),
		tokenLimitConfig: tokenLimitConfig,
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
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: "/orgs/" + orgSlug + "/admin/providers"},
				{Label: p.Name(), Href: "/orgs/" + orgSlug + "/admin/providers/" + string(p.ID()) + "/models"},
				{Label: m.ProxyName(), Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	templ.Handler(component.ModelForm(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) renderProviderFormError(w http.ResponseWriter, r *http.Request, ctx context.Context, user model.User, orgSlug string, org model.Organization, p model.Provider, isNew bool, errMsg string) {
	nav, footer := orgAdminNav(orgSlug)
	vmodel := component.ProviderFormVModel{
		Org:      org,
		Provider: p,
		IsNew:    isNew,
		Error:    errMsg,
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "org-" + orgSlug + "-providers",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Fournisseurs", Href: "/orgs/" + orgSlug + "/admin/providers"},
				{Label: p.Name(), Href: "/orgs/" + orgSlug + "/admin/providers/" + string(p.ID()) + "/models"},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	templ.Handler(component.ProviderForm(vmodel)).ServeHTTP(w, r)
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
