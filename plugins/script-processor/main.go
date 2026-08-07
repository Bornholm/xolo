package main

import "github.com/xolo-gateway/xolo/pkg/pluginsdk"

func main() {
	pluginsdk.ServeWithUI(&Plugin{}, "script-processor", newUIHandler())
}
