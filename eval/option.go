package eval

import "log/slog"

// Config holds options for [Run].
type Config struct {
	Concurrency int
	Logger      *slog.Logger
}

// Option configures an evaluation run.
type Option func(*Config)

// WithConcurrency sets the max parallel observation goroutines.
func WithConcurrency(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.Concurrency = n
		}
	}
}

// WithLogger sets the logger for the evaluation run.
func WithLogger(l *slog.Logger) Option {
	return func(c *Config) { c.Logger = l }
}
