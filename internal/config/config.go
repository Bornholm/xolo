package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type Config struct {
	Logger       Logger             `envPrefix:"LOGGER_"`
	HTTP         HTTP               `envPrefix:"HTTP_"`
	Storage      Storage            `envPrefix:"STORAGE_"`
	TaskRunner   TaskRunner         `envPrefix:"TASK_RUNNER_"`
	ExchangeRate ExchangeRateConfig `envPrefix:"EXCHANGE_RATE_"`
	// SecretKey is a 32-byte hex string used for AES-GCM encryption of provider API keys.
	SecretKey string `env:"SECRET_KEY"`
}

func Parse() (*Config, error) {
	conf, err := env.ParseAsWithOptions[Config](env.Options{
		Prefix: "XOLO_",
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := conf.Validate(); err != nil {
		return nil, errors.WithStack(err)
	}

	return &conf, nil
}

func (c *Config) Validate() error {
	if c.SecretKey == "" {
		return errors.New("XOLO_SECRET_KEY is required but not set (must be a 32-byte hex string, e.g. generated with: openssl rand -hex 32)")
	}

	return nil
}
