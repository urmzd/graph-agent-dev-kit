package agent

import (
	"strings"

	"github.com/urmzd/graph-agent-dev-kit/agent/core"
)

// StreamAggregator accumulates deltas into a complete Message.
type StreamAggregator interface {
	Push(delta core.Delta)
	Message() core.Message
	Reset()
}

// DefaultAggregator builds an AssistantMessage from streaming deltas.
type DefaultAggregator struct {
	contentBlocks []core.AssistantContent
	textBuf       strings.Builder
	inText        bool
	toolID        string
	toolName      string
	argsBuf       strings.Builder
	inTool        bool
}

// NewDefaultAggregator creates a new DefaultAggregator.
func NewDefaultAggregator() *DefaultAggregator {
	return &DefaultAggregator{}
}

func (a *DefaultAggregator) Push(d core.Delta) {
	switch v := d.(type) {
	case core.TextStartDelta:
		a.inText = true
		a.textBuf.Reset()
	case core.TextContentDelta:
		if a.inText {
			a.textBuf.WriteString(v.Content)
		}
	case core.TextEndDelta:
		if a.inText {
			a.contentBlocks = append(a.contentBlocks, core.TextContent{Text: a.textBuf.String()})
			a.inText = false
		}
	case core.ToolCallStartDelta:
		a.inTool = true
		a.toolID = v.ID
		a.toolName = v.Name
		a.argsBuf.Reset()
	case core.ToolCallArgumentDelta:
		if a.inTool {
			a.argsBuf.WriteString(v.Content)
		}
	case core.ToolCallEndDelta:
		if a.inTool {
			a.contentBlocks = append(a.contentBlocks, core.ToolUseContent{
				ID:        a.toolID,
				Name:      a.toolName,
				Arguments: v.Arguments,
			})
			a.inTool = false
		}
	}
}

func (a *DefaultAggregator) Message() core.Message {
	// Finalize any in-progress text
	blocks := make([]core.AssistantContent, len(a.contentBlocks))
	copy(blocks, a.contentBlocks)

	if a.inText && a.textBuf.Len() > 0 {
		blocks = append(blocks, core.TextContent{Text: a.textBuf.String()})
	}

	if len(blocks) == 0 {
		return nil
	}
	return core.AssistantMessage{Content: blocks}
}

func (a *DefaultAggregator) Reset() {
	a.contentBlocks = nil
	a.textBuf.Reset()
	a.inText = false
	a.toolID = ""
	a.toolName = ""
	a.argsBuf.Reset()
	a.inTool = false
}
