package ollama

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/urmzd/graph-agent-dev-kit/agent/core"
)

// Compile-time interface checks.
var (
	_ core.StructuredOutputProvider = (*Adapter)(nil)
	_ core.NamedProvider            = (*Adapter)(nil)
)

// Name implements core.NamedProvider.
func (a *Adapter) Name() string { return "ollama" }

// Adapter wraps the Ollama Client and implements core.Provider.
type Adapter struct {
	Client *Client
}

// NewAdapter creates a new Ollama Provider adapter.
func NewAdapter(client *Client) *Adapter {
	return &Adapter{Client: client}
}

// ChatStream implements core.Provider.
func (a *Adapter) ChatStream(ctx context.Context, messages []core.Message, tools []core.ToolDef) (<-chan core.Delta, error) {
	oMsgs := toOllamaMessages(messages)
	oTools := toOllamaTools(tools)

	rx, err := a.Client.ChatStream(ctx, oMsgs, oTools)
	if err != nil {
		return nil, &core.ProviderError{
			Provider: "ollama",
			Model:    a.Client.Model,
			Kind:     classifyOllamaError(err),
			Err:      err,
		}
	}

	return a.translateDeltas(rx), nil
}

// ChatStreamWithSchema implements core.StructuredOutputProvider.
func (a *Adapter) ChatStreamWithSchema(ctx context.Context, messages []core.Message, tools []core.ToolDef, schema *core.ParameterSchema) (<-chan core.Delta, error) {
	oMsgs := toOllamaMessages(messages)
	oTools := toOllamaTools(tools)

	var format any
	if schema != nil {
		format = parameterSchemaToMap(*schema)
	}

	rx, err := a.Client.ChatStreamWithFormat(ctx, oMsgs, oTools, format)
	if err != nil {
		return nil, &core.ProviderError{
			Provider: "ollama",
			Model:    a.Client.Model,
			Kind:     classifyOllamaError(err),
			Err:      err,
		}
	}

	return a.translateDeltas(rx), nil
}

// translateDeltas converts Ollama ChatChunk stream to core.Delta stream.
func (a *Adapter) translateDeltas(rx <-chan ChatChunk) <-chan core.Delta {
	out := make(chan core.Delta, 64)
	go func() {
		defer close(out)

		textStarted := false
		for chunk := range rx {
			if chunk.Done {
				if textStarted {
					out <- core.TextEndDelta{}
					textStarted = false
				}
				// Emit usage delta from the final chunk.
				out <- core.UsageDelta{
					PromptTokens:     chunk.PromptEvalCount,
					CompletionTokens: chunk.EvalCount,
					TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
				}
				continue
			}

			// Handle text content
			if chunk.Message.Content != "" {
				if !textStarted {
					out <- core.TextStartDelta{}
					textStarted = true
				}
				out <- core.TextContentDelta{Content: chunk.Message.Content}
			}

			// Handle tool calls
			if len(chunk.Message.ToolCalls) > 0 {
				if textStarted {
					out <- core.TextEndDelta{}
					textStarted = false
				}
				for _, tc := range chunk.Message.ToolCalls {
					id := core.NewID()
					out <- core.ToolCallStartDelta{ID: id, Name: tc.Function.Name}
					out <- core.ToolCallEndDelta{Arguments: tc.Function.Arguments}
				}
			}
		}

		if textStarted {
			out <- core.TextEndDelta{}
		}
	}()

	return out
}

// ContentSupport implements core.ContentNegotiator.
// Ollama supports JPEG and PNG natively via the images field.
func (a *Adapter) ContentSupport() core.ContentSupport {
	return core.ContentSupport{
		NativeTypes: map[core.MediaType]bool{
			core.MediaJPEG: true,
			core.MediaPNG:  true,
		},
	}
}

// ── Convenience methods (not part of Provider) ──────────────────────

// Generate delegates to the underlying client.
func (a *Adapter) Generate(ctx context.Context, prompt string) (string, error) {
	return a.Client.Generate(ctx, prompt)
}

// GenerateWithModel delegates to the underlying client.
func (a *Adapter) GenerateWithModel(ctx context.Context, prompt, model string, format, options any) (string, error) {
	return a.Client.GenerateWithModel(ctx, prompt, model, format, options)
}

// GenerateStream delegates to the underlying client.
func (a *Adapter) GenerateStream(ctx context.Context, prompt string) (<-chan string, error) {
	return a.Client.GenerateStream(ctx, prompt)
}

// Embed delegates to the underlying client.
func (a *Adapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return a.Client.Embed(ctx, text)
}

// ── Conversion helpers ──────────────────────────────────────────────

func toOllamaMessages(msgs []core.Message) []ChatMessage {
	out := make([]ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		switch v := m.(type) {
		case core.SystemMessage:
			// Split: text goes to system role, tool results go to tool role.
			var textParts []string
			var toolResults []core.ToolResultContent
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					textParts = append(textParts, bc.Text)
				case core.ToolResultContent:
					toolResults = append(toolResults, bc)
				}
			}
			if len(textParts) > 0 {
				out = append(out, ChatMessage{Role: "system", Content: strings.Join(textParts, "")})
			}
			for _, tr := range toolResults {
				out = append(out, ChatMessage{Role: "tool", Content: tr.Text})
			}
		case core.UserMessage:
			// Split: text goes to user role, tool results go to tool role.
			var textParts []string
			var images []string
			var toolResults []core.ToolResultContent
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					textParts = append(textParts, bc.Text)
				case core.ToolResultContent:
					toolResults = append(toolResults, bc)
				case core.FileContent:
					if bc.Data != nil {
						images = append(images, base64.StdEncoding.EncodeToString(bc.Data))
					}
				}
			}
			if len(textParts) > 0 || len(images) > 0 {
				out = append(out, ChatMessage{
					Role:    "user",
					Content: strings.Join(textParts, ""),
					Images:  images,
				})
			}
			for _, tr := range toolResults {
				out = append(out, ChatMessage{Role: "tool", Content: tr.Text})
			}
		case core.AssistantMessage:
			msg := ChatMessage{Role: "assistant"}
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					msg.Content += bc.Text
				case core.ToolUseContent:
					msg.ToolCalls = append(msg.ToolCalls, ToolCall{
						Function: ToolCallFunction{
							Name:      bc.Name,
							Arguments: bc.Arguments,
						},
					})
				}
			}
			out = append(out, msg)
		}
	}
	return out
}

func toOllamaTools(defs []core.ToolDef) []Tool {
	out := make([]Tool, len(defs))
	for i, d := range defs {
		props := make(map[string]ToolProperty, len(d.Parameters.Properties))
		for k, v := range d.Parameters.Properties {
			props[k] = convertProperty(v)
		}
		out[i] = Tool{
			Type: "function",
			Function: ToolFunction{
				Name:        d.Name,
				Description: d.Description,
				Parameters: ToolFunctionParams{
					Type:       d.Parameters.Type,
					Required:   d.Parameters.Required,
					Properties: props,
				},
			},
		}
	}
	return out
}

// convertProperty recursively converts a core.PropertyDef to an Ollama ToolProperty.
func convertProperty(p core.PropertyDef) ToolProperty {
	tp := ToolProperty{
		Type:        p.Type,
		Description: p.Description,
		Enum:        p.Enum,
		Required:    p.Required,
		Default:     p.Default,
	}
	if p.Items != nil {
		items := convertProperty(*p.Items)
		tp.Items = &items
	}
	if len(p.Properties) > 0 {
		tp.Properties = make(map[string]ToolProperty, len(p.Properties))
		for k, v := range p.Properties {
			tp.Properties[k] = convertProperty(v)
		}
	}
	return tp
}

// parameterSchemaToMap converts a ParameterSchema to a map for the Ollama format field.
func parameterSchemaToMap(ps core.ParameterSchema) map[string]any {
	schema := map[string]any{"type": ps.Type}
	if len(ps.Required) > 0 {
		schema["required"] = ps.Required
	}
	if len(ps.Properties) > 0 {
		props := make(map[string]any, len(ps.Properties))
		for k, v := range ps.Properties {
			props[k] = propertyDefToMap(v)
		}
		schema["properties"] = props
	}
	return schema
}

func propertyDefToMap(p core.PropertyDef) map[string]any {
	m := map[string]any{"type": p.Type}
	if p.Description != "" {
		m["description"] = p.Description
	}
	if len(p.Enum) > 0 {
		m["enum"] = p.Enum
	}
	if p.Default != nil {
		m["default"] = p.Default
	}
	if p.Items != nil {
		m["items"] = propertyDefToMap(*p.Items)
	}
	if len(p.Properties) > 0 {
		props := make(map[string]any, len(p.Properties))
		for k, v := range p.Properties {
			props[k] = propertyDefToMap(v)
		}
		m["properties"] = props
	}
	if len(p.Required) > 0 {
		m["required"] = p.Required
	}
	return m
}

// classifyOllamaError inspects the error to determine if it's transient.
func classifyOllamaError(err error) core.ErrorKind {
	s := err.Error()
	if strings.Contains(s, "connection refused") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "returned 5") ||
		strings.Contains(s, "returned 429") {
		return core.ErrorKindTransient
	}
	return core.ErrorKindPermanent
}
