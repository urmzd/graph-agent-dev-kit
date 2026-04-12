package types

import "context"

// Provider is the narrow LLM interface the agent loop needs.
// Model selection is handled via ConfigContent in the message tree,
// not as a parameter — each provider uses its own configured default.
type Provider interface {
	ChatStream(ctx context.Context, messages []Message, tools []ToolDef) (<-chan Delta, error)
}

// NamedProvider is an optional interface providers can implement
// for identification in logs and error messages.
type NamedProvider interface {
	Provider
	Name() string
}

// StructuredOutputProvider is an optional interface for providers that support
// constraining LLM output to a JSON schema.
type StructuredOutputProvider interface {
	Provider
	ChatStreamWithSchema(ctx context.Context, messages []Message, tools []ToolDef, schema *ParameterSchema) (<-chan Delta, error)
}

// ModelProvider is an optional interface providers can implement
// to expose the configured model name for telemetry and logging.
type ModelProvider interface {
	Provider
	Model() string
}

// Closer is an optional interface providers can implement for graceful shutdown.
type Closer interface {
	Close() error
}

// ProviderName returns the name of a provider if it implements NamedProvider,
// otherwise returns "unknown".
func ProviderName(p Provider) string {
	if np, ok := p.(NamedProvider); ok {
		return np.Name()
	}
	return "unknown"
}

// ProviderModel returns the model of a provider if it implements ModelProvider,
// otherwise returns an empty string.
func ProviderModel(p Provider) string {
	if mp, ok := p.(ModelProvider); ok {
		return mp.Model()
	}
	return ""
}

// CloseProvider closes a provider if it implements Closer, otherwise returns nil.
func CloseProvider(p Provider) error {
	if c, ok := p.(Closer); ok {
		return c.Close()
	}
	return nil
}
