package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/Padiwa/go-safe/pkg/modelstore"
	"github.com/bornholm/xolo/pkg/pluginsdk"
)

type pluginUI struct {
	plugin *Plugin
}

func newUIHandler() http.Handler {
	ui := &pluginUI{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", ui.handleIndex)
	mux.HandleFunc("POST /api/config", ui.handleSaveConfig)
	return mux
}

type uiPageData struct {
	BasePath    string
	OrgID       string
	Config      Config
	Success     bool
	Error       string
	ModelStatus []modelStatusEntry
}

type modelStatusEntry struct {
	Name   string
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
			slog.WarnContext(ctx, "go-safe-classifier/ui: failed to load config", slog.Any("error", err))
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

func (ui *pluginUI) fetchModelStatus(ctx context.Context, cfg Config) []modelStatusEntry {
	var opts []modelstore.Option
	if cfg.CacheDir != "" {
		opts = append(opts, modelstore.WithCacheDir(cfg.CacheDir))
	}
	if cfg.ManifestURL != "" {
		opts = append(opts, modelstore.WithManifestURL(cfg.ManifestURL))
	}
	if cfg.Offline {
		opts = append(opts, modelstore.WithOfflineMode(true))
	}

	store, err := modelstore.New(opts...)
	if err != nil {
		slog.WarnContext(ctx, "go-safe-classifier/ui: failed to create model store", slog.Any("error", err))
		return nil
	}

	names, err := store.Available(ctx)
	if err != nil {
		slog.WarnContext(ctx, "go-safe-classifier/ui: failed to list available models", slog.Any("error", err))
		return nil
	}

	entries := make([]modelStatusEntry, 0, len(names))
	for _, n := range names {
		entries = append(entries, modelStatusEntry{
			Name:   n,
			Cached: store.IsCached(n),
		})
	}
	return entries
}

func (ui *pluginUI) handleIndex(w http.ResponseWriter, r *http.Request) {
	pd, err := ui.loadPageData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func renderTempl(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("go-safe-classifier/ui: template render error", slog.Any("error", err))
	}
}