package pipeline

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/dist
var distFS embed.FS

// NewAssetsHandler returns an http.Handler serving the compiled React Flow bundle.
func NewAssetsHandler() http.Handler {
	sub, err := fs.Sub(distFS, "assets/dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
