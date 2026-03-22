package anthropic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/urmzd/saige/agent/core"
)

// Compile-time interface checks.
var (
	_ core.StructuredOutputProvider = (*Adapter)(nil)
	_ core.NamedProvider            = (*Adapter)(nil)
)

// Adapter wraps the official Anthropic SDK client and implements core.Provider,
// core.NamedProvider, core.StructuredOutputProvider, and core.ContentNegotiator.
type Adapter struct {
	client    anthropic.Client
	model     anthropic.Model
	maxTokens int64
}

// Option configures the Anthropic adapter.
type Option func(*Adapter)

// WithMaxTokens sets the max tokens for responses.
func WithMaxTokens(n int64) Option {
	return func(a *Adapter) { a.maxTokens = n }
}

// NewAdapter creates a new Anthropic provider adapter using the official SDK.
func NewAdapter(apiKey, model string, opts ...Option) *Adapter {
	a := &Adapter{
		client:    anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:     anthropic.Model(model),
		maxTokens: 4096,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Name implements core.NamedProvider.
func (a *Adapter) Name() string { return "anthropic" }

// ChatStream implements core.Provider.
func (a *Adapter) ChatStream(ctx context.Context, messages []core.Message, tools []core.ToolDef) (<-chan core.Delta, error) {
	systemBlocks, aMsgs := toAnthropicParams(messages)
	aTools := toAnthropicTools(tools)

	params := anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		Messages:  aMsgs,
		System:    systemBlocks,
	}
	if len(aTools) > 0 {
		params.Tools = aTools
	}

	stream := a.client.Messages.NewStreaming(ctx, params)
	return a.consumeStream(stream, nil), nil
}

// ChatStreamWithSchema implements core.StructuredOutputProvider.
// Anthropic has no native response_format; we inject a hidden tool and force the model to call it.
func (a *Adapter) ChatStreamWithSchema(ctx context.Context, messages []core.Message, tools []core.ToolDef, schema *core.ParameterSchema) (<-chan core.Delta, error) {
	systemBlocks, aMsgs := toAnthropicParams(messages)
	aTools := toAnthropicTools(tools)

	params := anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		Messages:  aMsgs,
		System:    systemBlocks,
	}

	if schema != nil {
		// Inject a hidden tool whose input schema is the desired response schema.
		props := make(map[string]any, len(schema.Properties))
		for k, v := range schema.Properties {
			props[k] = propertyToSchema(v)
		}
		hiddenTool := anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        "structured_output",
				Description: anthropic.String("Return the structured response"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: props,
				},
			},
		}
		aTools = append(aTools, hiddenTool)
		params.ToolChoice = anthropic.ToolChoiceParamOfTool("structured_output")
	}

	if len(aTools) > 0 {
		params.Tools = aTools
	}

	isStructured := func(name string) bool {
		return schema != nil && name == "structured_output"
	}

	stream := a.client.Messages.NewStreaming(ctx, params)
	return a.consumeStream(stream, isStructured), nil
}

// consumeStream reads from the Anthropic streaming response and emits deltas.
// If isStructuredTool is non-nil and returns true for a tool_use block name,
// the tool's input JSON is emitted as text deltas instead of tool call deltas.
func (a *Adapter) consumeStream(stream *ssestream.Stream[anthropic.MessageStreamEventUnion], isStructuredTool func(string) bool) <-chan core.Delta {
	out := make(chan core.Delta, 64)
	go func() {
		defer close(out)

		var currentBlockType string
		var currentBlockName string
		var toolArgsBuf []byte

		for stream.Next() {
			evt := stream.Current()

			switch evt.Type {
			case "message_start":
				if evt.Message.Usage.InputTokens > 0 {
					out <- core.UsageDelta{
						PromptTokens: int(evt.Message.Usage.InputTokens),
						TotalTokens:  int(evt.Message.Usage.InputTokens + evt.Message.Usage.OutputTokens),
					}
				}

			case "content_block_start":
				currentBlockType = evt.ContentBlock.Type
				currentBlockName = evt.ContentBlock.Name
				switch evt.ContentBlock.Type {
				case "text":
					out <- core.TextStartDelta{}
				case "tool_use":
					toolArgsBuf = toolArgsBuf[:0]
					if isStructuredTool != nil && isStructuredTool(evt.ContentBlock.Name) {
						// Emit as text for structured output.
						out <- core.TextStartDelta{}
					} else {
						out <- core.ToolCallStartDelta{
							ID:   evt.ContentBlock.ID,
							Name: evt.ContentBlock.Name,
						}
					}
				}

			case "content_block_delta":
				switch evt.Delta.Type {
				case "text_delta":
					out <- core.TextContentDelta{Content: evt.Delta.Text}
				case "input_json_delta":
					toolArgsBuf = append(toolArgsBuf, evt.Delta.PartialJSON...)
					if isStructuredTool != nil && isStructuredTool(currentBlockName) {
						out <- core.TextContentDelta{Content: evt.Delta.PartialJSON}
					} else {
						out <- core.ToolCallArgumentDelta{Content: evt.Delta.PartialJSON}
					}
				}

			case "content_block_stop":
				switch currentBlockType {
				case "text":
					out <- core.TextEndDelta{}
				case "tool_use":
					if isStructuredTool != nil && isStructuredTool(currentBlockName) {
						out <- core.TextEndDelta{}
					} else {
						var args map[string]any
						if len(toolArgsBuf) > 0 {
							_ = json.Unmarshal(toolArgsBuf, &args)
						}
						out <- core.ToolCallEndDelta{Arguments: args}
					}
				}
				currentBlockType = ""
				currentBlockName = ""

			case "message_delta":
				if evt.Usage.OutputTokens > 0 {
					out <- core.UsageDelta{
						CompletionTokens: int(evt.Usage.OutputTokens),
						TotalTokens:      int(evt.Usage.OutputTokens),
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			out <- core.ErrorDelta{Error: classifyAnthropicError(err)}
		}
	}()

	return out
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

func toAnthropicParams(msgs []core.Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam) {
	var system []anthropic.TextBlockParam
	var out []anthropic.MessageParam

	for _, m := range msgs {
		switch v := m.(type) {
		case core.SystemMessage:
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					system = append(system, anthropic.TextBlockParam{Text: bc.Text})
				case core.ToolResultContent:
					block := anthropic.NewToolResultBlock(bc.ToolCallID, bc.Text, false)
					out = appendMsg(out, "user", block)
				}
			}

		case core.UserMessage:
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					out = appendMsg(out, "user", anthropic.NewTextBlock(bc.Text))
				case core.ToolResultContent:
					out = appendMsg(out, "user", anthropic.NewToolResultBlock(bc.ToolCallID, bc.Text, false))
				case core.FileContent:
					if bc.Data != nil && isImageType(bc.MediaType) {
						b64 := base64.StdEncoding.EncodeToString(bc.Data)
						out = appendMsg(out, "user", anthropic.NewImageBlockBase64(string(bc.MediaType), b64))
					} else if bc.Data != nil {
						out = appendMsg(out, "user", anthropic.NewTextBlock("[File: "+bc.Filename+"] "+string(bc.Data)))
					}
				}
			}

		case core.AssistantMessage:
			for _, c := range v.Content {
				switch bc := c.(type) {
				case core.TextContent:
					out = appendMsg(out, "assistant", anthropic.NewTextBlock(bc.Text))
				case core.ToolUseContent:
					out = appendMsg(out, "assistant", anthropic.NewToolUseBlock(bc.ID, bc.Arguments, bc.Name))
				}
			}
		}
	}

	return system, out
}

// appendMsg appends a content block to the last message if same role, otherwise creates new.
func appendMsg(msgs []anthropic.MessageParam, role string, block anthropic.ContentBlockParamUnion) []anthropic.MessageParam {
	r := anthropic.MessageParamRole(role)
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == r {
		msgs[len(msgs)-1].Content = append(msgs[len(msgs)-1].Content, block)
		return msgs
	}
	return append(msgs, anthropic.MessageParam{
		Role:    r,
		Content: []anthropic.ContentBlockParamUnion{block},
	})
}

func isImageType(mt core.MediaType) bool {
	switch mt {
	case core.MediaJPEG, core.MediaPNG, core.MediaGIF, core.MediaWebP:
		return true
	}
	return false
}

func toAnthropicTools(defs []core.ToolDef) []anthropic.ToolUnionParam {
	if len(defs) == 0 {
		return nil
	}
	out := make([]anthropic.ToolUnionParam, len(defs))
	for i, d := range defs {
		props := make(map[string]any, len(d.Parameters.Properties))
		for k, v := range d.Parameters.Properties {
			props[k] = propertyToSchema(v)
		}
		out[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        d.Name,
				Description: anthropic.String(d.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: props,
				},
			},
		}
	}
	return out
}

func propertyToSchema(p core.PropertyDef) map[string]any {
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
		m["items"] = propertyToSchema(*p.Items)
	}
	if len(p.Properties) > 0 {
		nested := make(map[string]any, len(p.Properties))
		for k, v := range p.Properties {
			nested[k] = propertyToSchema(v)
		}
		m["properties"] = nested
	}
	if len(p.Required) > 0 {
		m["required"] = p.Required
	}
	return m
}

func classifyAnthropicError(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return &core.ProviderError{
			Provider: "anthropic",
			Kind:     core.ClassifyHTTPStatus(apiErr.StatusCode),
			Code:     apiErr.StatusCode,
			Err:      err,
		}
	}
	return &core.ProviderError{
		Provider: "anthropic",
		Kind:     core.ErrorKindPermanent,
		Err:      err,
	}
}
