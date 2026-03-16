package config

// PluginsConfig controls where Xolo looks for plugin binaries.
type PluginsConfig struct {
	// Dir is the directory scanned at startup for executable plugin binaries.
	// Set via XOLO_PLUGINS_DIR. An empty or missing directory is not an error.
	Dir string `env:"DIR" envDefault:"./plugins"`
}
