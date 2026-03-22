package types

import (
	"context"
	"strings"
)

// Compactor reduces message history to fit context windows.
type Compactor interface {
	Compact(ctx context.Context, messages []Message, provider Provider) ([]Message, error)
}

// ── Data-driven compaction config ────────────────────────────────────

// CompactStrategy names a compaction algorithm.
type CompactStrategy string

const (
	CompactNone          CompactStrategy = "none"
	CompactSlidingWindow CompactStrategy = "sliding_window"
	CompactSummarize     CompactStrategy = "summarize"
)

// CompactConfig is a serialisable description of a compaction strategy.
type CompactConfig struct {
	Strategy   CompactStrategy
	WindowSize int // for sliding_window
	Threshold  int // for summarize
}

// ToCompactor converts the config into a Compactor implementation.
func (cc CompactConfig) ToCompactor() Compactor {
	switch cc.Strategy {
	case CompactSlidingWindow:
		return NewSlidingWindowCompactor(cc.WindowSize)
	case CompactSummarize:
		return NewSummarizeCompactor(cc.Threshold)
	default:
		return NoopCompactor{}
	}
}

// NoopCompactor passes messages through unchanged.
type NoopCompactor struct{}

func (NoopCompactor) Compact(_ context.Context, messages []Message, _ Provider) ([]Message, error) {
	return messages, nil
}

// SlidingWindowCompactor keeps the first message (system) and the last N messages.
type SlidingWindowCompactor struct {
	WindowSize int
}

func NewSlidingWindowCompactor(n int) *SlidingWindowCompactor {
	return &SlidingWindowCompactor{WindowSize: n}
}

func (c *SlidingWindowCompactor) Compact(_ context.Context, messages []Message, _ Provider) ([]Message, error) {
	if len(messages) <= c.WindowSize+1 {
		return messages, nil
	}
	// Keep first (system) + last N
	result := make([]Message, 0, c.WindowSize+1)
	result = append(result, messages[0])
	result = append(result, messages[len(messages)-c.WindowSize:]...)
	return result, nil
}

// SummarizeCompactor summarizes older messages when history exceeds a threshold.
type SummarizeCompactor struct {
	Threshold int
}

func NewSummarizeCompactor(threshold int) *SummarizeCompactor {
	return &SummarizeCompactor{Threshold: threshold}
}

func (c *SummarizeCompactor) Compact(ctx context.Context, messages []Message, provider Provider) ([]Message, error) {
	if len(messages) <= c.Threshold {
		return messages, nil
	}

	// Summarize all but last 4 messages using the provider
	keepLast := 4
	if keepLast > len(messages)-1 {
		keepLast = len(messages) - 1
	}

	toSummarize := messages[1 : len(messages)-keepLast]
	if len(toSummarize) == 0 {
		return messages, nil
	}

	// Build summary prompt
	summaryReq := []Message{
		NewSystemMessage("Summarize the following conversation concisely, preserving key facts and decisions."),
		NewUserMessage(MessagesToText(toSummarize)),
	}

	rx, err := provider.ChatStream(ctx, summaryReq, nil)
	if err != nil {
		return messages, nil // fallback: no compaction
	}

	var sb strings.Builder
	for delta := range rx {
		if tc, ok := delta.(TextContentDelta); ok {
			sb.WriteString(tc.Content)
		}
	}
	summary := sb.String()

	result := make([]Message, 0, keepLast+2)
	result = append(result, messages[0]) // system
	result = append(result, NewUserMessage("Previous conversation summary: "+summary))
	result = append(result, messages[len(messages)-keepLast:]...)
	return result, nil
}

// MessagesToText converts messages to a plain-text representation.
func MessagesToText(msgs []Message) string {
	var b strings.Builder
	for _, m := range msgs {
		switch v := m.(type) {
		case SystemMessage:
			for _, c := range v.Content {
				switch bc := c.(type) {
				case TextContent:
					b.WriteString("System: ")
					b.WriteString(bc.Text)
					b.WriteByte('\n')
				case ToolResultContent:
					b.WriteString("Tool Result [")
					b.WriteString(bc.ToolCallID)
					b.WriteString("]: ")
					b.WriteString(bc.Text)
					b.WriteByte('\n')
				}
			}
		case UserMessage:
			for _, c := range v.Content {
				switch bc := c.(type) {
				case TextContent:
					b.WriteString("User: ")
					b.WriteString(bc.Text)
					b.WriteByte('\n')
				case ToolResultContent:
					b.WriteString("Tool Result [")
					b.WriteString(bc.ToolCallID)
					b.WriteString("]: ")
					b.WriteString(bc.Text)
					b.WriteByte('\n')
				case FileContent:
					b.WriteString("User: [file: ")
					b.WriteString(bc.Filename)
					b.WriteString(" (")
					b.WriteString(string(bc.MediaType))
					b.WriteString(")]\n")
				}
			}
		case AssistantMessage:
			for _, c := range v.Content {
				if tc, ok := c.(TextContent); ok {
					b.WriteString("Assistant: ")
					b.WriteString(tc.Text)
					b.WriteByte('\n')
				}
			}
		}
	}
	return b.String()
}
