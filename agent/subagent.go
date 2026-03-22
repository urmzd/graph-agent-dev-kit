package agent

import (
	"context"

	"github.com/urmzd/saige/agent/types"
)

// SubAgentDef defines a sub-agent that can be delegated to.
type SubAgentDef struct {
	Name         string
	Description  string
	SystemPrompt string
	Provider     types.Provider
	Tools        *types.ToolRegistry
	SubAgents    []SubAgentDef // sub-agents can have their own sub-agents
	MaxIter      int
}

// SubAgentInvoker is implemented by tools that wrap a sub-agent.
// The agent loop checks for this interface to enable delta forwarding
// instead of opaque Execute().
type SubAgentInvoker interface {
	InvokeAgent(ctx context.Context, task string) *EventStream
}

// subAgentTool wraps a sub-agent as a tool. It implements both types.Tool and
// SubAgentInvoker so the agent loop can forward child deltas.
type subAgentTool struct {
	def     types.ToolDef
	factory func() *Agent
}

func (t *subAgentTool) Definition() types.ToolDef { return t.def }

// Execute provides a blocking fallback — runs the child agent and returns
// the concatenated text. The agent loop prefers InvokeAgent for streaming.
func (t *subAgentTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	task, _ := args["task"].(string)
	stream := t.InvokeAgent(ctx, task)
	var result string
	for d := range stream.Deltas() {
		if tc, ok := d.(types.TextContentDelta); ok {
			result += tc.Content
		}
	}
	return result, stream.Wait()
}

// InvokeAgent creates a fresh child agent and invokes it, returning its stream.
func (t *subAgentTool) InvokeAgent(ctx context.Context, task string) *EventStream {
	child := t.factory()
	return child.Invoke(ctx, []types.Message{types.NewUserMessage(task)})
}
