package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/templ"
	fuzzydsl "github.com/bornholm/go-fuzzy/dsl"
	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/bornholm/xolo/plugins/smart-model/complexity"
	"github.com/bornholm/xolo/plugins/smart-model/complexity/data"
)

// newUIHandler builds the multi-tab HTTP UI for the smart-model plugin.
func newUIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tab/trigger", http.StatusFound)
	})
	mux.HandleFunc("GET /tab/{tab}", handleTab)
	mux.HandleFunc("POST /api/config", handleSaveConfig)
	mux.HandleFunc("POST /api/test", handleTestRequest)
	return mux
}

// ── Shared page data ──────────────────────────────────────────────────────────

type uiPageData struct {
	BasePath      string
	OrgID         string
	ActiveTab     string
	Config        Config
	Categories    []string
	LogLines      []string
	AllModels     []*proto.ModelInfo
	VirtualModels []*proto.ModelInfo
}

func loadPageData(r *http.Request, activeTab string) (uiPageData, error) {
	ctx := r.Context()

	orgID := r.Header.Get("X-Xolo-Org-Id")
	basePath := r.Header.Get("X-Xolo-Plugin-Base-Path")
	if basePath == "" {
		basePath = "/"
	}

	host := pluginsdk.HostClientFromContext(ctx)
	pluginName := pluginsdk.PluginNameFromContext(ctx)

	var raw string
	if host != nil && orgID != "" {
		var err error
		raw, err = host.GetConfig(ctx, orgID, pluginName)
		if err != nil {
			slog.WarnContext(ctx, "smart-model/ui: failed to load config", slog.Any("error", err))
		}
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		return uiPageData{}, err
	}

	// Load classifier categories.
	var categories []string
	nb, err := complexity.LoadModel(data.RawModel)
	if err == nil {
		categories = nb.Classes
	}

	// Load log lines (last 50).
	var logLines []string
	if cfg.LogEnabled && cfg.LogPath != "" {
		logLines = readLastLines(cfg.LogPath, 50)
	}

	// Partition models into real and virtual.
	var allModels, virtualModels []*proto.ModelInfo
	if host != nil && orgID != "" {
		if all, err := host.ListModels(ctx, orgID); err == nil {
			for _, m := range all {
				if m.IsVirtual {
					virtualModels = append(virtualModels, m)
				} else {
					allModels = append(allModels, m)
				}
			}
		}
	}

	return uiPageData{
		BasePath:      basePath,
		OrgID:         orgID,
		ActiveTab:     activeTab,
		Config:        cfg,
		Categories:    categories,
		LogLines:      logLines,
		AllModels:     allModels,
		VirtualModels: virtualModels,
	}, nil
}

// ── Tab handler ───────────────────────────────────────────────────────────────

func handleTab(w http.ResponseWriter, r *http.Request) {
	activeTab := r.PathValue("tab")
	if activeTab == "" {
		activeTab = "trigger"
	}
	pd, err := loadPageData(r, activeTab)
	if err != nil {
		httpErr(w, err)
		return
	}
	renderTempl(w, r, page(pd))
}

// ── API handlers ──────────────────────────────────────────────────────────────

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get("X-Xolo-Org-Id")
	host := pluginsdk.HostClientFromContext(ctx)
	pluginName := pluginsdk.PluginNameFromContext(ctx)

	if host == nil || orgID == "" {
		http.Error(w, "missing host or org context", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	raw, _ := host.GetConfig(ctx, orgID, pluginName)
	cfg, err := ParseConfig(raw)
	if err != nil {
		cfg = DefaultConfig()
	}

	if rules := r.FormValue("rules"); rules != "" {
		if _, err := fuzzydsl.ParseRulesAndVariables(rules); err != nil {
			http.Error(w, fmt.Sprintf("Règles invalides : %v", err), http.StatusBadRequest)
			return
		}
		cfg.Rules = rules
	}

	if es := r.FormValue("energy_sensitivity"); es != "" {
		var v float64
		if _, err := fmt.Sscanf(es, "%f", &v); err == nil {
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			cfg.EnergySensitivity = v
		}
	}

	cfg.LogEnabled = r.FormValue("log_enabled") == "true"
	if lp := r.FormValue("log_path"); lp != "" {
		cfg.LogPath = lp
	}

	// Add or update a model config.
	if proxyName := r.FormValue("model_proxy_name"); proxyName != "" {
		enabled := r.FormValue("model_enabled") != "false"
		mc := ModelConfig{ProxyName: proxyName, Enabled: enabled}
		if plStr := r.FormValue("model_power_level"); plStr != "" {
			var v float64
			if _, err := fmt.Sscanf(plStr, "%f", &v); err == nil && v >= 0 && v <= 1 {
				mc.PowerLevelOverride = &v
			}
		}
		for _, cat := range r.Form["model_categories"] {
			if cat != "" {
				mc.Categories = append(mc.Categories, cat)
			}
		}
		found := false
		for i, m := range cfg.Models {
			if m.ProxyName == proxyName {
				cfg.Models[i] = mc
				found = true
				break
			}
		}
		if !found {
			cfg.Models = append(cfg.Models, mc)
		}
	}

	// Update trigger models (from tab=trigger form only).
	if r.FormValue("tab") == "trigger" {
		cfg.TriggerModels = r.Form["trigger_models"]
	}

	// Remove a model config.
	if removeProxy := r.FormValue("remove_model"); removeProxy != "" {
		filtered := cfg.Models[:0]
		for _, m := range cfg.Models {
			if m.ProxyName != removeProxy {
				filtered = append(filtered, m)
			}
		}
		cfg.Models = filtered
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := host.SaveConfig(ctx, orgID, pluginName, string(cfgJSON)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tab := r.FormValue("tab")
	if tab == "" {
		tab = "rules"
	}
	http.Redirect(w, r, "/tab/"+tab, http.StatusFound)
}

type testResult struct {
	Vars              InputVars `json:"vars"`
	DesiredPowerLevel float64   `json:"desired_power_level"`
	SelectedModel     string    `json:"selected_model,omitempty"`
	Error             string    `json:"error,omitempty"`
}

func handleTestRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get("X-Xolo-Org-Id")
	host := pluginsdk.HostClientFromContext(ctx)
	pluginName := pluginsdk.PluginNameFromContext(ctx)

	var raw string
	if host != nil && orgID != "" {
		raw, _ = host.GetConfig(ctx, orgID, pluginName)
	}
	cfg, _ := ParseConfig(raw)

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prompt := r.FormValue("prompt")
	var messagesJSON string
	if prompt != "" {
		msgs := []map[string]string{{"role": "user", "content": prompt}}
		b, _ := json.Marshal(msgs)
		messagesJSON = string(b)
	}

	vars := ScoreRequest(messagesJSON, "", nil, cfg)
	desiredPL, err := runFuzzyInference(cfg.Rules, vars, cfg.EnergySensitivity)

	result := testResult{
		Vars:              vars,
		DesiredPowerLevel: desiredPL,
	}
	if err != nil {
		result.Error = err.Error()
	}

	// Try to select the best model using the available models from the host.
	if err == nil && host != nil && orgID != "" {
		if models, listErr := host.ListModels(ctx, orgID); listErr == nil {
			result.SelectedModel = selectModel(models, desiredPL, vars, cfg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func readLastLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

func httpErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func renderTempl(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("smart-model/ui: template render error", slog.Any("error", err))
	}
}

