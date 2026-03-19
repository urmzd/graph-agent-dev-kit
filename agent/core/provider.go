package core

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

// ProviderName returns the name of a provider if it implements NamedProvider,
// otherwise returns "unknown".
func ProviderName(p Provider) string {
	if np, ok := p.(NamedProvider); ok {
		return np.Name()
	}
	return "unknown"
}
