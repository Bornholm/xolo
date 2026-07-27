package main

import "github.com/bornholm/xolo/pkg/pluginsdk"

func main() {
	pluginsdk.ServeWithUI(newPlugin(), "go-safe-classifier", newUIHandler())
}