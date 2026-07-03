package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/xolo/pkg/pluginsdk"
)

func newUIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("POST /api/config", handleSaveConfig)
	return mux
}

type uiPageData struct {
	BasePath string
	OrgID    string
	Config   Config
	Success  bool
}

func loadPageData(r *http.Request) uiPageData {
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
		slog.Info("system-prompt GetConfig", slog.String("orgID", orgID), slog.String("pluginName", pluginName), slog.String("raw", raw), slog.Any("err", err))
		if err != nil {
			slog.WarnContext(ctx, "system-prompt/ui: failed to load config", slog.Any("error", err))
		}
	}

	cfg := parseConfig(raw)
	return uiPageData{BasePath: basePath, OrgID: orgID, Config: cfg}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	pd := loadPageData(r)
	pd.Success = r.URL.Query().Get("saved") == "1"
	renderTempl(w, r, page(pd))
}

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get("X-Xolo-Org-Id")
	host := pluginsdk.HostClientFromContext(ctx)
	pluginName := pluginsdk.PluginNameFromContext(ctx)

	if host == nil || orgID == "" {
		slog.Error("system-prompt: SaveConfig failed", slog.Any("error", http.StatusBadRequest))
		http.Error(w, "missing host or org context", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := Config{
		SystemPrompt: r.FormValue("system_prompt"),
		Append:       r.FormValue("append") == "true",
	}
	slog.Error("system-prompt: SystemPrompt", slog.Any("error", cfg.SystemPrompt))

	b, _ := json.Marshal(cfg)
	if err := host.SaveConfig(ctx, orgID, pluginName, string(b)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/?saved=1", http.StatusFound)
}

func renderTempl(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		slog.Error("system-prompt/ui: render error", slog.Any("error", err))
	}
}
