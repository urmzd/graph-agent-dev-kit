// Package agenttest provides testing utilities for the agent SDK.
package agenttest

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/urmzd/saige/agent/core"
)

// ScriptedProvider replays predefined delta sequences, one per ChatStream call.
// Thread-safe for concurrent use.
type ScriptedProvider struct {
	mu        sync.Mutex
	call      int
	Responses [][]core.Delta
}

func (p *ScriptedProvider) ChatStream(_ context.Context, _ []core.Message, _ []core.ToolDef) (<-chan core.Delta, error) {
	p.mu.Lock()
	idx := p.call
	p.call++
	p.mu.Unlock()

	ch := make(chan core.Delta, 64)
	go func() {
		defer close(ch)
		if idx < len(p.Responses) {
			for _, d := range p.Responses[idx] {
				ch <- d
			}
		}
	}()
	return ch, nil
}

// TextResponse creates a delta sequence for a simple text response.
func TextResponse(text string) []core.Delta {
	return []core.Delta{
		core.TextStartDelta{},
		core.TextContentDelta{Content: text},
		core.TextEndDelta{},
	}
}

// ToolCallResponse creates a delta sequence for a tool call.
func ToolCallResponse(id, name string, args map[string]any) []core.Delta {
	return []core.Delta{
		core.ToolCallStartDelta{ID: id, Name: name},
		core.ToolCallEndDelta{Arguments: args},
	}
}

// CollectDeltas drains a delta channel into a slice.
func CollectDeltas(ch <-chan core.Delta) []core.Delta {
	var deltas []core.Delta
	for d := range ch {
		deltas = append(deltas, d)
	}
	return deltas
}

// CollectText drains a delta channel and returns concatenated text content.
func CollectText(ch <-chan core.Delta) string {
	var sb strings.Builder
	for d := range ch {
		if tc, ok := d.(core.TextContentDelta); ok {
			sb.WriteString(tc.Content)
		}
	}
	return sb.String()
}

// CollectToolCalls drains a delta channel and returns all completed tool calls.
func CollectToolCalls(ch <-chan core.Delta) []core.ToolUseContent {
	var calls []core.ToolUseContent
	var currentID, currentName string
	for d := range ch {
		switch v := d.(type) {
		case core.ToolCallStartDelta:
			currentID = v.ID
			currentName = v.Name
		case core.ToolCallEndDelta:
			calls = append(calls, core.ToolUseContent{
				ID:        currentID,
				Name:      currentName,
				Arguments: v.Arguments,
			})
		}
	}
	return calls
}

// AssertTextContains verifies the delta channel produces text containing substr.
func AssertTextContains(t *testing.T, ch <-chan core.Delta, substr string) {
	t.Helper()
	text := CollectText(ch)
	if !strings.Contains(text, substr) {
		t.Errorf("expected text to contain %q, got %q", substr, text)
	}
}

// AssertToolCalled verifies a specific tool was called in the deltas.
func AssertToolCalled(t *testing.T, deltas []core.Delta, name string) {
	t.Helper()
	for _, d := range deltas {
		if v, ok := d.(core.ToolCallStartDelta); ok && v.Name == name {
			return
		}
	}
	t.Errorf("expected tool %q to be called, but it was not", name)
}

// AssertNoErrors verifies no error deltas were emitted.
func AssertNoErrors(t *testing.T, deltas []core.Delta) {
	t.Helper()
	for _, d := range deltas {
		if v, ok := d.(core.ErrorDelta); ok {
			t.Errorf("unexpected error delta: %v", v.Error)
		}
	}
}

// AssertDone verifies a DoneDelta was emitted.
func AssertDone(t *testing.T, deltas []core.Delta) {
	t.Helper()
	for _, d := range deltas {
		if _, ok := d.(core.DoneDelta); ok {
			return
		}
	}
	t.Error("expected DoneDelta but none was found")
}

// MockTool is a test tool with configurable behavior.
type MockTool struct {
	Def    core.ToolDef
	Result string
	Err    error
	Calls  []map[string]any // records all calls made
	mu     sync.Mutex
}

func (t *MockTool) Definition() core.ToolDef { return t.Def }

func (t *MockTool) Execute(_ context.Context, args map[string]any) (string, error) {
	t.mu.Lock()
	t.Calls = append(t.Calls, args)
	t.mu.Unlock()
	return t.Result, t.Err
}

// CallCount returns the number of times the tool was called.
func (t *MockTool) CallCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.Calls)
}
