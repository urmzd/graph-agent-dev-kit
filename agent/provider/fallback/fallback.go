package fallback

import (
	"context"

	"github.com/urmzd/saige/agent/core"
)

// Provider tries providers in order, falling back on failure.
// By default it falls back on any error. Set FallbackOn to control
// which errors trigger fallback (e.g. core.IsTransient for transient-only).
type Provider struct {
	Providers  []core.Provider
	FallbackOn func(error) bool // nil = fallback on any error
}

// New creates a provider that tries each in order.
func New(providers ...core.Provider) *Provider {
	return &Provider{Providers: providers}
}

func (f *Provider) Name() string { return "fallback" }

func (f *Provider) ChatStream(ctx context.Context, messages []core.Message, tools []core.ToolDef) (<-chan core.Delta, error) {
	shouldFallback := f.FallbackOn
	if shouldFallback == nil {
		shouldFallback = func(error) bool { return true }
	}

	var errs []error
	for _, p := range f.Providers {
		ch, err := p.ChatStream(ctx, messages, tools)
		if err == nil {
			return ch, nil
		}
		errs = append(errs, err)

		if ctx.Err() != nil {
			break
		}
		if !shouldFallback(err) {
			break
		}
	}

	return nil, &core.FallbackError{Errors: errs}
}

// ChatStreamWithSchema implements core.StructuredOutputProvider.
// For each provider, it tries ChatStreamWithSchema if the provider supports it,
// otherwise falls back to ChatStream.
func (f *Provider) ChatStreamWithSchema(ctx context.Context, messages []core.Message, tools []core.ToolDef, schema *core.ParameterSchema) (<-chan core.Delta, error) {
	shouldFallback := f.FallbackOn
	if shouldFallback == nil {
		shouldFallback = func(error) bool { return true }
	}

	var errs []error
	for _, p := range f.Providers {
		var ch <-chan core.Delta
		var err error

		if sp, ok := p.(core.StructuredOutputProvider); ok {
			ch, err = sp.ChatStreamWithSchema(ctx, messages, tools, schema)
		} else {
			ch, err = p.ChatStream(ctx, messages, tools)
		}

		if err == nil {
			return ch, nil
		}
		errs = append(errs, err)

		if ctx.Err() != nil {
			break
		}
		if !shouldFallback(err) {
			break
		}
	}

	return nil, &core.FallbackError{Errors: errs}
}
