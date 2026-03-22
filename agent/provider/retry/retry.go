package retry

import (
	"context"
	"math"
	"time"

	"github.com/urmzd/saige/agent/types"
)

// Config controls retry behavior.
type Config struct {
	MaxAttempts int           // total attempts (1 = no retry)
	BaseDelay   time.Duration // initial delay between retries
	MaxDelay    time.Duration // cap on delay
	Multiplier  float64       // backoff multiplier (default 2.0)
	ShouldRetry func(error) bool // nil = retry on IsTransient errors
}

// DefaultConfig returns sensible defaults: 3 attempts, 500ms base,
// 10s cap, 2x exponential backoff, transient-only.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
	}
}

// Provider wraps a Provider with retry logic and exponential backoff.
type Provider struct {
	Inner  types.Provider
	Config Config
}

// New wraps a provider with the given retry config.
func New(inner types.Provider, cfg Config) *Provider {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 500 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 10 * time.Second
	}
	return &Provider{Inner: inner, Config: cfg}
}

func (r *Provider) Name() string {
	return "retry(" + types.ProviderName(r.Inner) + ")"
}

func (r *Provider) ChatStream(ctx context.Context, messages []types.Message, tools []types.ToolDef) (<-chan types.Delta, error) {
	return r.retryLoop(ctx, func() (<-chan types.Delta, error) {
		return r.Inner.ChatStream(ctx, messages, tools)
	})
}

// ChatStreamWithSchema implements types.StructuredOutputProvider.
// If the inner provider supports structured output, retries use it.
// Otherwise, falls back to ChatStream (schema is lost).
func (r *Provider) ChatStreamWithSchema(ctx context.Context, messages []types.Message, tools []types.ToolDef, schema *types.ParameterSchema) (<-chan types.Delta, error) {
	if sp, ok := r.Inner.(types.StructuredOutputProvider); ok {
		return r.retryLoop(ctx, func() (<-chan types.Delta, error) {
			return sp.ChatStreamWithSchema(ctx, messages, tools, schema)
		})
	}
	return r.ChatStream(ctx, messages, tools)
}

// retryLoop runs the call function with exponential backoff.
func (r *Provider) retryLoop(ctx context.Context, call func() (<-chan types.Delta, error)) (<-chan types.Delta, error) {
	shouldRetry := r.Config.ShouldRetry
	if shouldRetry == nil {
		shouldRetry = types.IsTransient
	}

	var lastErr error
	for attempt := range r.Config.MaxAttempts {
		ch, err := call()
		if err == nil {
			return ch, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return nil, lastErr
		}
		if !shouldRetry(err) {
			return nil, lastErr
		}

		// Backoff before next attempt (skip after last attempt).
		if attempt < r.Config.MaxAttempts-1 {
			delay := time.Duration(float64(r.Config.BaseDelay) * math.Pow(r.Config.Multiplier, float64(attempt)))
			if delay > r.Config.MaxDelay {
				delay = r.Config.MaxDelay
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, &types.RetryError{Attempts: r.Config.MaxAttempts, Last: lastErr}
}
