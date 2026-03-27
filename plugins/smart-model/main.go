package main

import (
	"net/http"

	"github.com/bornholm/xolo/pkg/pluginsdk"
)

func main() {
	pluginsdk.ServeWithUI(&Plugin{}, PluginName, buildUIHandler())
}

// buildUIHandler returns the HTTP handler for the plugin's web UI.
// It is defined in ui_handler.go.
func buildUIHandler() http.Handler {
	return newUIHandler()
}
