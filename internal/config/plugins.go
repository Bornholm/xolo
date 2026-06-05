package config

// PluginsConfig controls where Xolo looks for plugin binaries.
type PluginsConfig struct {
	// Dir is the directory scanned at startup for executable plugin binaries.
	// Set via XOLO_PLUGINS_DIR. An empty or missing directory is not an error.
	Dir string `env:"DIR" envDefault:"./plugins"`

	// MemLimit sets the GOMEMLIMIT environment variable for each plugin subprocess,
	// making the Go GC more aggressive before reaching the limit.
	// Set via XOLO_PLUGINS_MEM_LIMIT. Accepts Go-style suffixes: "512MiB", "1GiB",
	// or raw bytes. Empty string disables the limit (default).
	MemLimit string `env:"MEM_LIMIT" envDefault:""`
}
