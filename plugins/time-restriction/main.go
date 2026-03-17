package main

import (
	_ "time/tzdata" // embed IANA timezone database for production containers without /usr/share/zoneinfo

	"github.com/bornholm/xolo/pkg/pluginsdk"
)

func main() {
	pluginsdk.Serve(&Plugin{})
}
