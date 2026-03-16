package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/bornholm/go-fuzzy/dsl"
	"github.com/bornholm/xolo/pkg/pluginsdk"
)

//go:embed templates
var templateFS embed.FS

var configTmpl = template.Must(
	template.ParseFS(templateFS, "templates/config.html"),
)

type templateData struct {
	Config       Config
	Error        string
	SignalsJSON  template.JS
	ModelsJSON   template.JS
	MappingsJSON template.JS
	UIMode       template.JS
}

type uiHandler struct{}

func newUIHandler() http.Handler { return &uiHandler{} }

func (h *uiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *uiHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := loadConfigFromContext(r.Context(), r)
	if err != nil {
		slog.ErrorContext(r.Context(), "fuzzy-model-selector ui: load config", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	renderConfigPage(w, cfg, "")
}

func (h *uiHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	cfg, validErr := buildConfigFromForm(r)
	if validErr != nil {
		renderConfigPage(w, cfg, validErr.Error())
		return
	}

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		renderConfigPage(w, cfg, "Erreur interne lors de la sérialisation.")
		return
	}

	client := pluginsdk.HostClientFromContext(ctx)
	if client == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	orgID := r.Header.Get("X-Xolo-Org-Id")
	pluginName := pluginsdk.PluginNameFromContext(ctx)

	if err := client.SaveConfig(ctx, orgID, pluginName, string(configJSON)); err != nil {
		slog.ErrorContext(ctx, "fuzzy-model-selector ui: save config", slog.Any("error", err))
		renderConfigPage(w, cfg, "Erreur lors de la sauvegarde : "+err.Error())
		return
	}

	http.Redirect(w, r, "./", http.StatusSeeOther)
}

func loadConfigFromContext(ctx context.Context, r *http.Request) (Config, error) {
	client := pluginsdk.HostClientFromContext(ctx)
	if client == nil {
		return Config{VirtualModel: "auto", UIMode: "simple"}, nil
	}
	orgID := r.Header.Get("X-Xolo-Org-Id")
	pluginName := pluginsdk.PluginNameFromContext(ctx)
	cfgJSON, err := client.GetConfig(ctx, orgID, pluginName)
	if err != nil {
		return Config{}, fmt.Errorf("get config: %w", err)
	}
	return parseConfig(cfgJSON)
}

func buildConfigFromForm(r *http.Request) (Config, error) {
	cfg := Config{}
	cfg.VirtualModel = r.FormValue("virtual_model")
	if cfg.VirtualModel == "" {
		cfg.VirtualModel = "auto"
	}
	if bp, err := strconv.ParseFloat(r.FormValue("budget_preference"), 64); err == nil {
		cfg.BudgetPreference = bp
	}
	cfg.UIMode = r.FormValue("ui_mode")
	if cfg.UIMode == "" {
		cfg.UIMode = "simple"
	}
	if sigJSON := r.FormValue("signals_json"); sigJSON != "" {
		_ = json.Unmarshal([]byte(sigJSON), &cfg.Signals)
	}
	if mdlJSON := r.FormValue("models_json"); mdlJSON != "" {
		_ = json.Unmarshal([]byte(mdlJSON), &cfg.Models)
	}
	if mpJSON := r.FormValue("simple_mappings_json"); mpJSON != "" {
		_ = json.Unmarshal([]byte(mpJSON), &cfg.SimpleMappings)
	}
	if cfg.UIMode == "simple" {
		cfg.Rules = GenerateDSL(cfg.SimpleMappings)
	} else {
		cfg.Rules = r.FormValue("rules")
	}
	if cfg.Rules != "" {
		if _, err := dsl.ParseRulesAndVariables(cfg.Rules); err != nil {
			return cfg, fmt.Errorf("Règles DSL invalides : %w", err)
		}
	}
	return cfg, nil
}

func renderConfigPage(w http.ResponseWriter, cfg Config, errMsg string) {
	signalsJSON, _ := json.Marshal(cfg.Signals)
	modelsJSON, _ := json.Marshal(cfg.Models)
	mappingsJSON, _ := json.Marshal(cfg.SimpleMappings)
	uiMode := cfg.UIMode
	if uiMode == "" {
		uiMode = "simple"
	}
	data := templateData{
		Config:       cfg,
		Error:        errMsg,
		SignalsJSON:  template.JS(signalsJSON),
		ModelsJSON:   template.JS(modelsJSON),
		MappingsJSON: template.JS(mappingsJSON),
		UIMode:       template.JS(`"` + uiMode + `"`),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := configTmpl.ExecuteTemplate(w, "config.html", data); err != nil {
		slog.Error("fuzzy-model-selector ui: template render error", slog.Any("error", err))
	}
}
