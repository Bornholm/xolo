package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-anon/pkg/modelstore"
	"github.com/bornholm/xolo/pkg/pluginsdk"
)

type pluginUI struct {
	plugin *Plugin
}

func newUIHandler(p *Plugin) http.Handler {
	ui := &pluginUI{plugin: p}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", ui.handleIndex)
	mux.HandleFunc("POST /api/config", ui.handleSaveConfig)
	return mux
}

// uiPageData holds all data needed to render the configuration page.
type uiPageData struct {
	BasePath    string
	OrgID       string
	Config      Config
	Success     bool
	Error       string
	ModelStatus []modelStatusEntry
}

type modelStatusEntry struct {
	Lang   string
	Cached bool
}

func (ui *pluginUI) loadPageData(r *http.Request) (uiPageData, error) {
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
			slog.WarnContext(ctx, "pseudonymizer/ui: failed to load config", slog.Any("error", err))
		}
	}

	cfg, err := parseConfig(raw)
	if err != nil {
		cfg = defaultConfig()
	}

	pd := uiPageData{
		BasePath: basePath,
		OrgID:    orgID,
		Config:   cfg,
	}

	pd.ModelStatus = ui.fetchModelStatus(ctx, cfg)
	return pd, nil
}

// fetchModelStatus builds a lightweight model status list (no downloads).
func (ui *pluginUI) fetchModelStatus(ctx context.Context, cfg Config) []modelStatusEntry {
	opts := storeOptionsFromConfig(cfg)
	store, err := modelstore.New(opts...)
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer/ui: failed to create model store", slog.Any("error", err))
		return nil
	}

	langs, err := store.Available(ctx)
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer/ui: failed to list available models", slog.Any("error", err))
		return nil
	}

	entries := make([]modelStatusEntry, 0, len(langs))
	for _, lang := range langs {
		entries = append(entries, modelStatusEntry{
			Lang:   lang,
			Cached: store.IsCached(lang),
		})
	}
	return entries
}

func (ui *pluginUI) handleIndex(w http.ResponseWriter, r *http.Request) {
	pd, err := ui.loadPageData(r)
	if err != nil {
		httpError(w, err)
		return
	}
	pd.Success = r.URL.Query().Get("saved") == "1"
	renderTempl(w, r, page(pd))
}

func (ui *pluginUI) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
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

	cfg := configFromForm(r)

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := host.SaveConfig(ctx, orgID, pluginName, string(cfgJSON)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/?saved=1", http.StatusFound)
}

// configFromForm builds a Config from form values.
func configFromForm(r *http.Request) Config {
	cfg := Config{
		CacheDir:              r.FormValue("cache_dir"),
		ManifestURL:           r.FormValue("manifest_url"),
		Offline:               r.FormValue("offline") == "on",
		Language:              r.FormValue("language"),
		Strategy:              r.FormValue("strategy"),
		FirstNameReclassify:   r.FormValue("first_name_reclassify") == "on",
		Merge:                 r.FormValue("merge") == "on",
		NameCompletion:        r.FormValue("name_completion") == "on",
		BuiltinRegexPatterns:  r.FormValue("builtin_regex_patterns") == "on",
		BuiltinSecretPatterns: r.FormValue("builtin_secret_patterns") == "on",
	}

	if minConf := r.FormValue("min_confidence"); minConf != "" {
		var f float64
		if err := json.Unmarshal([]byte(minConf), &f); err == nil {
			cfg.MinConfidence = f
		}
	}
	if maxTok := r.FormValue("max_tokens"); maxTok != "" {
		var n int
		if err := json.Unmarshal([]byte(maxTok), &n); err == nil {
			cfg.MaxTokens = n
		}
	}

	if cfg.Language == "" {
		cfg.Language = "fr"
	}
	if cfg.Strategy == "" {
		cfg.Strategy = "tag"
	}

	return cfg
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func httpError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func renderTempl(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("pseudonymizer/ui: template render error", slog.Any("error", err))
	}
}
