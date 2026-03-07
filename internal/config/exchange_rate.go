package config

import "time"

type ExchangeRateConfig struct {
	Provider        string        `env:"PROVIDER" envDefault:"frankfurter"` // "frankfurter" or "file"
	FilePath        string        `env:"FILE_PATH"`
	TTL             time.Duration `env:"TTL" envDefault:"24h"`
	RefreshInterval time.Duration `env:"REFRESH_INTERVAL" envDefault:"24h"`
}
