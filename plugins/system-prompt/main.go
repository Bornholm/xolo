package main

import "github.com/xolo-gateway/xolo/pkg/pluginsdk"

func main() {
	p := &Plugin{}
	pluginsdk.ServeWithUI(p, "system-prompt", newUIHandler())
}
