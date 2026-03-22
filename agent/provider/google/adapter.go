package google

import (
	"context"

	"github.com/urmzd/saige/agent/core"
	"google.golang.org/genai"
)

// Compile-time interface checks.
var (
	_ core.StructuredOutputProvider = (*Adapter)(nil)
	_ core.NamedProvider            = (*Adapter)(nil)
)

// Adapter wraps the official Google GenAI SDK client and implements core.Provider,
// core.NamedProvider, core.StructuredOutputProvider, and core.ContentNegotiator.
type Adapter struct {
	client *genai.Client
	model  string
}

// NewAdapter creates a new Google provider adapter using the official SDK.
func NewAdapter(ctx context.Context, apiKey, model string) (*Adapter, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &Adapter{client: client, model: model}, nil
}

// Name implements core.NamedProvider.
func (a *Adapter) Name() string { return "google" }

// ChatStream implements core.Provider.
func (a *Adapter) ChatStream(ctx context.Context, messages []core.Message, tools []core.ToolDef) (<-chan core.Delta, error) {
	systemInst, contents := toGeminiContents(messages)
	config := &genai.GenerateContentConfig{}
	if systemInst != nil {
		config.SystemInstruction = systemInst
	}

	gTools := toGeminiTools(tools)
	if len(gTools) > 0 {
		config.Tools = gTools
	}

	return a.chatStream(ctx, contents, config)
}

// ChatStreamWithSchema implements core.StructuredOutputProvider.
func (a *Adapter) ChatStreamWithSchema(ctx context.Context, messages []core.Message, tools []core.ToolDef, schema *core.ParameterSchema) (<-chan core.Delta, error) {
	systemInst, contents := toGeminiContents(messages)
	config := &genai.GenerateContentConfig{}
	if systemInst != nil {
		config.SystemInstruction = systemInst
	}

	gTools := toGeminiTools(tools)
	if len(gTools) > 0 {
		config.Tools = gTools
	}

	if schema != nil {
		config.ResponseMIMEType = "application/json"
		config.ResponseSchema = parameterSchemaToGemini(*schema)
	}

	return a.chatStream(ctx, contents, config)
}

// chatStream runs the streaming generation goroutine.
func (a *Adapter) chatStream(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (<-chan core.Delta, error) {
	out := make(chan core.Delta, 64)
	go func() {
		defer close(out)

		for resp, err := range a.client.Models.GenerateContentStream(ctx, a.model, contents, config) {
			if err != nil {
				out <- core.ErrorDelta{Error: &core.ProviderError{
					Provider: "google",
					Model:    a.model,
					Kind:     core.ErrorKindPermanent,
					Err:      err,
				}}
				return
			}

			// Emit text content.
			if text := resp.Text(); text != "" {
				out <- core.TextStartDelta{}
				out <- core.TextContentDelta{Content: text}
				out <- core.TextEndDelta{}
			}

			// Emit function calls (Gemini sends complete calls per chunk).
			for _, fc := range resp.FunctionCalls() {
				id := fc.ID
				if id == "" {
					id = core.NewID()
				}
				out <- core.ToolCallStartDelta{ID: id, Name: fc.Name}
				out <- core.ToolCallEndDelta{Arguments: fc.Args}
			}

			// Emit usage.
			if resp.UsageMetadata != nil {
				out <- core.UsageDelta{
					PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
					CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
					TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
				}
			}
		}
	}()

	return out, nil
}

// ContentSupport implements core.ContentNegotiator.
func (a *Adapter) ContentSupport() core.ContentSupport {
	return core.ContentSupport{
		NativeTypes: map[core.MediaType]bool{
			core.MediaJPEG: true,
			core.MediaPNG:  true,
			core.MediaGIF:  true,
			core.MediaWebP: true,
			core.MediaPDF:  true,
		},
	}
}

// ── Conversion helpers ──────────────────────────────────────────────

func toGeminiContents(msgs []core.Message) (*genai.Content, []*genai.Content) {
	var systemParts []*genai.Part
	var contents []*genai.Content

	for _, m := range msgs {
		switch v := m.(type) {
		case core.SystemMessage:
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					systemParts = append(systemParts, &genai.Part{Text: bc.Text})
				case core.ToolResultContent:
					// Tool results go as function responses from "user" role.
					contents = append(contents, genai.NewContentFromFunctionResponse(
						bc.ToolCallID, map[string]any{"result": bc.Text}, "user",
					))
				}
			}

		case core.UserMessage:
			var parts []*genai.Part
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					parts = append(parts, &genai.Part{Text: bc.Text})
				case core.ToolResultContent:
					contents = append(contents, genai.NewContentFromFunctionResponse(
						bc.ToolCallID, map[string]any{"result": bc.Text}, "user",
					))
				case core.FileContent:
					if bc.Data != nil {
						parts = append(parts, &genai.Part{
							InlineData: &genai.Blob{
								Data:     bc.Data,
								MIMEType: string(bc.MediaType),
							},
						})
					}
				}
			}
			if len(parts) > 0 {
				contents = append(contents, genai.NewContentFromParts(parts, "user"))
			}

		case core.AssistantMessage:
			var parts []*genai.Part
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					parts = append(parts, &genai.Part{Text: bc.Text})
				case core.ToolUseContent:
					parts = append(parts, &genai.Part{
						FunctionCall: &genai.FunctionCall{
							Name: bc.Name,
							Args: bc.Arguments,
						},
					})
				}
			}
			if len(parts) > 0 {
				contents = append(contents, genai.NewContentFromParts(parts, "model"))
			}
		}
	}

	var systemInst *genai.Content
	if len(systemParts) > 0 {
		systemInst = &genai.Content{Parts: systemParts}
	}
	return systemInst, contents
}

func toGeminiTools(defs []core.ToolDef) []*genai.Tool {
	if len(defs) == 0 {
		return nil
	}
	funcs := make([]*genai.FunctionDeclaration, len(defs))
	for i, d := range defs {
		funcs[i] = &genai.FunctionDeclaration{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  parameterSchemaToGemini(d.Parameters),
		}
	}
	return []*genai.Tool{{FunctionDeclarations: funcs}}
}

func parameterSchemaToGemini(ps core.ParameterSchema) *genai.Schema {
	s := &genai.Schema{
		Type:     mapType(ps.Type),
		Required: ps.Required,
	}
	if len(ps.Properties) > 0 {
		s.Properties = make(map[string]*genai.Schema, len(ps.Properties))
		for k, v := range ps.Properties {
			s.Properties[k] = propertyToGemini(v)
		}
	}
	return s
}

func propertyToGemini(p core.PropertyDef) *genai.Schema {
	s := &genai.Schema{
		Type:        mapType(p.Type),
		Description: p.Description,
		Enum:        p.Enum,
		Required:    p.Required,
		Default:     p.Default,
	}
	if p.Items != nil {
		s.Items = propertyToGemini(*p.Items)
	}
	if len(p.Properties) > 0 {
		s.Properties = make(map[string]*genai.Schema, len(p.Properties))
		for k, v := range p.Properties {
			s.Properties[k] = propertyToGemini(v)
		}
	}
	return s
}

func mapType(t string) genai.Type {
	switch t {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeString
	}
}
