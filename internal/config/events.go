package config

import "time"

// EventsConfig configures the event system: the ring-buffer retention bounds,
// the periodic alert evaluation/purge cadence and the async emitter buffer.
type EventsConfig struct {
	// MaxPerOrg is the global hard cap on non-pinned events retained per org.
	MaxPerOrg int `env:"MAX_PER_ORG" envDefault:"100000"`
	// DefaultPerOrg is the retention applied to orgs without an explicit override.
	DefaultPerOrg int `env:"DEFAULT_PER_ORG" envDefault:"10000"`
	// EvaluationInterval is how often alerts are evaluated.
	EvaluationInterval time.Duration `env:"EVALUATION_INTERVAL" envDefault:"30s"`
	// PurgeInterval is how often the ring-buffer purge runs.
	PurgeInterval time.Duration `env:"PURGE_INTERVAL" envDefault:"5m"`
	// EmitBufferSize is the size of the async emitter channel.
	EmitBufferSize int `env:"EMIT_BUFFER_SIZE" envDefault:"1024"`
}
