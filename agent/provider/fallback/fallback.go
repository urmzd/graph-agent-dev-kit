package fallback

import (
	"context"

	"github.com/urmzd/saige/agent/types"
)

// Provider tries providers in order, falling back on failure.
// By default it falls back on any error. Set FallbackOn to control
// which errors trigger fallback (e.g. types.IsTransient for transient-only).
type Provider struct {
	Providers  []types.Provider
	FallbackOn func(error) bool // nil = fallback on any error
}

// New creates a provider that tries each in order.
func New(providers ...types.Provider) *Provider {
	return &Provider{Providers: providers}
}

func (f *Provider) Name() string { return "fallback" }

func (f *Provider) ChatStream(ctx context.Context, messages []types.Message, tools []types.ToolDef) (<-chan types.Delta, error) {
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

	return nil, &types.FallbackError{Errors: errs}
}

// ChatStreamWithSchema implements types.StructuredOutputProvider.
// For each provider, it tries ChatStreamWithSchema if the provider supports it,
// otherwise falls back to ChatStream.
func (f *Provider) ChatStreamWithSchema(ctx context.Context, messages []types.Message, tools []types.ToolDef, schema *types.ParameterSchema) (<-chan types.Delta, error) {
	shouldFallback := f.FallbackOn
	if shouldFallback == nil {
		shouldFallback = func(error) bool { return true }
	}

	var errs []error
	for _, p := range f.Providers {
		var ch <-chan types.Delta
		var err error

		if sp, ok := p.(types.StructuredOutputProvider); ok {
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

	return nil, &types.FallbackError{Errors: errs}
}
