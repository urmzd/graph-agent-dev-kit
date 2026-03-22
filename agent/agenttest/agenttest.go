// Package agenttest provides testing utilities for the agent SDK.
package agenttest

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/urmzd/saige/agent/types"
)

// ScriptedProvider replays predefined delta sequences, one per ChatStream call.
// Thread-safe for concurrent use.
type ScriptedProvider struct {
	mu        sync.Mutex
	call      int
	Responses [][]types.Delta
}

func (p *ScriptedProvider) ChatStream(_ context.Context, _ []types.Message, _ []types.ToolDef) (<-chan types.Delta, error) {
	p.mu.Lock()
	idx := p.call
	p.call++
	p.mu.Unlock()

	ch := make(chan types.Delta, 64)
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
func TextResponse(text string) []types.Delta {
	return []types.Delta{
		types.TextStartDelta{},
		types.TextContentDelta{Content: text},
		types.TextEndDelta{},
	}
}

// ToolCallResponse creates a delta sequence for a tool call.
func ToolCallResponse(id, name string, args map[string]any) []types.Delta {
	return []types.Delta{
		types.ToolCallStartDelta{ID: id, Name: name},
		types.ToolCallEndDelta{Arguments: args},
	}
}

// CollectDeltas drains a delta channel into a slice.
func CollectDeltas(ch <-chan types.Delta) []types.Delta {
	var deltas []types.Delta
	for d := range ch {
		deltas = append(deltas, d)
	}
	return deltas
}

// CollectText drains a delta channel and returns concatenated text content.
func CollectText(ch <-chan types.Delta) string {
	var sb strings.Builder
	for d := range ch {
		if tc, ok := d.(types.TextContentDelta); ok {
			sb.WriteString(tc.Content)
		}
	}
	return sb.String()
}

// CollectToolCalls drains a delta channel and returns all completed tool calls.
func CollectToolCalls(ch <-chan types.Delta) []types.ToolUseContent {
	var calls []types.ToolUseContent
	var currentID, currentName string
	for d := range ch {
		switch v := d.(type) {
		case types.ToolCallStartDelta:
			currentID = v.ID
			currentName = v.Name
		case types.ToolCallEndDelta:
			calls = append(calls, types.ToolUseContent{
				ID:        currentID,
				Name:      currentName,
				Arguments: v.Arguments,
			})
		}
	}
	return calls
}

// AssertTextContains verifies the delta channel produces text containing substr.
func AssertTextContains(t *testing.T, ch <-chan types.Delta, substr string) {
	t.Helper()
	text := CollectText(ch)
	if !strings.Contains(text, substr) {
		t.Errorf("expected text to contain %q, got %q", substr, text)
	}
}

// AssertToolCalled verifies a specific tool was called in the deltas.
func AssertToolCalled(t *testing.T, deltas []types.Delta, name string) {
	t.Helper()
	for _, d := range deltas {
		if v, ok := d.(types.ToolCallStartDelta); ok && v.Name == name {
			return
		}
	}
	t.Errorf("expected tool %q to be called, but it was not", name)
}

// AssertNoErrors verifies no error deltas were emitted.
func AssertNoErrors(t *testing.T, deltas []types.Delta) {
	t.Helper()
	for _, d := range deltas {
		if v, ok := d.(types.ErrorDelta); ok {
			t.Errorf("unexpected error delta: %v", v.Error)
		}
	}
}

// AssertDone verifies a DoneDelta was emitted.
func AssertDone(t *testing.T, deltas []types.Delta) {
	t.Helper()
	for _, d := range deltas {
		if _, ok := d.(types.DoneDelta); ok {
			return
		}
	}
	t.Error("expected DoneDelta but none was found")
}

// MockTool is a test tool with configurable behavior.
type MockTool struct {
	Def    types.ToolDef
	Result string
	Err    error
	Calls  []map[string]any // records all calls made
	mu     sync.Mutex
}

func (t *MockTool) Definition() types.ToolDef { return t.Def }

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
