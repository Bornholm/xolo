package main

import "github.com/bornholm/xolo/pkg/pluginsdk"

func main() {
	p := &Plugin{}
	pluginsdk.ServeWithUI(p, "system-prompt", newUIHandler())
}
