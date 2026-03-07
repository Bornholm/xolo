package common

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/*
var assetsFS embed.FS

type Handler struct {
	mux *http.ServeMux
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler() *Handler {
	handler := &Handler{
		mux: http.NewServeMux(),
	}

	assets, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		panic(err)
	}

	handler.mux.Handle("GET /", http.FileServerFS(assets))

	return handler
}

var _ http.Handler = &Handler{}
