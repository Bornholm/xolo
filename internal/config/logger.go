package config

type Logger struct {
	Level int `env:"LEVEL,expand" envDefault:"0"`
}
