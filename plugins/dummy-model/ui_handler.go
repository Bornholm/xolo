package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

func newUIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("POST /api/config", handleSaveConfig)
	return mux
}

// ── Shared page data ──────────────────────────────────────────────────────────

type uiPageData struct {
	BasePath      string
	OrgID         string
	Config        Config
	VirtualModels []*proto.ModelInfo
}

func loadPageData(r *http.Request) (uiPageData, error) {
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
			slog.WarnContext(ctx, "dummy-model/ui: failed to load config", slog.Any("error", err))
		}
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		return uiPageData{}, err
	}

	var virtualModels []*proto.ModelInfo
	if host != nil && orgID != "" {
		if all, err := host.ListModels(ctx, orgID); err == nil {
			for _, m := range all {
				if m.IsVirtual {
					virtualModels = append(virtualModels, m)
				}
			}
		}
	}

	return uiPageData{
		BasePath:      basePath,
		OrgID:         orgID,
		Config:        cfg,
		VirtualModels: virtualModels,
	}, nil
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleIndex(w http.ResponseWriter, r *http.Request) {
	pd, err := loadPageData(r)
	if err != nil {
		httpErr(w, err)
		return
	}
	renderTempl(w, r, page(pd))
}

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
		cfg = Config{}
	}

	cfg.TriggerModels = r.Form["trigger_models"]

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := host.SaveConfig(ctx, orgID, pluginName, string(cfgJSON)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func httpErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func renderTempl(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("dummy-model/ui: template render error", slog.Any("error", err))
	}
}
