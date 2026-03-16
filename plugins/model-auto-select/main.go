package main

import (
	"github.com/bornholm/xolo/pkg/pluginsdk"
)

func main() {
	pluginsdk.ServeWithUI(&Plugin{}, "model-auto-select", newUIHandler())
}
