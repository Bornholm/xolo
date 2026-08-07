package main

import (
	_ "time/tzdata" // embed IANA timezone database for production containers without /usr/share/zoneinfo

	"github.com/xolo-gateway/xolo/pkg/pluginsdk"
)

func main() {
	pluginsdk.ServeWithUI(&Plugin{}, "time-restriction", newUIHandler())
}
