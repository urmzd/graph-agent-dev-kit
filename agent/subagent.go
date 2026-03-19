package agent

import (
	"context"

	"github.com/urmzd/graph-agent-dev-kit/agent/core"
)

// SubAgentDef defines a sub-agent that can be delegated to.
type SubAgentDef struct {
	Name         string
	Description  string
	SystemPrompt string
	Provider     core.Provider
	Tools        *core.ToolRegistry
	SubAgents    []SubAgentDef // sub-agents can have their own sub-agents
	MaxIter      int
}

// SubAgentInvoker is implemented by tools that wrap a sub-agent.
// The agent loop checks for this interface to enable delta forwarding
// instead of opaque Execute().
type SubAgentInvoker interface {
	InvokeAgent(ctx context.Context, task string) *EventStream
}

// subAgentTool wraps a sub-agent as a tool. It implements both core.Tool and
// SubAgentInvoker so the agent loop can forward child deltas.
type subAgentTool struct {
	def     core.ToolDef
	factory func() *Agent
}

func (t *subAgentTool) Definition() core.ToolDef { return t.def }

// Execute provides a blocking fallback — runs the child agent and returns
// the concatenated text. The agent loop prefers InvokeAgent for streaming.
func (t *subAgentTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	task, _ := args["task"].(string)
	stream := t.InvokeAgent(ctx, task)
	var result string
	for d := range stream.Deltas() {
		if tc, ok := d.(core.TextContentDelta); ok {
			result += tc.Content
		}
	}
	return result, stream.Wait()
}

// InvokeAgent creates a fresh child agent and invokes it, returning its stream.
func (t *subAgentTool) InvokeAgent(ctx context.Context, task string) *EventStream {
	child := t.factory()
	return child.Invoke(ctx, []core.Message{core.NewUserMessage(task)})
}
