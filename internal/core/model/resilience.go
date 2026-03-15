package model

import "time"

// RetryConfig configures the retry middleware for an LLM provider.
// Delay must be > 0 when Enabled.
type RetryConfig struct {
	Enabled     bool          `json:"enabled"`
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"` // stored as nanoseconds
}

// RateLimitConfig configures the token-bucket rate limiter for an LLM provider.
// Interval must be > 0 when Enabled. MaxBurst is the burst capacity.
type RateLimitConfig struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"` // stored as nanoseconds
	MaxBurst int           `json:"max_burst"`
}

// TokenLimitConfig configures the chat-completion token throughput limiter.
// Interval must be > 0 when Enabled (used as denominator in rate calculation).
type TokenLimitConfig struct {
	Enabled   bool          `json:"enabled"`
	MaxTokens int           `json:"max_tokens"`
	Interval  time.Duration `json:"interval"` // stored as nanoseconds
}
