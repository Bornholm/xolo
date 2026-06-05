package main

import "github.com/bornholm/xolo/pkg/pluginsdk"

func main() {
	p := newPlugin()
	pluginsdk.ServeWithUI(p, "pseudonymizer", newUIHandler(p))
}
