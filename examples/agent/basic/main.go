// Package main demonstrates a simple chat agent with a single "add" tool
// that adds two numbers. It creates an Ollama-backed agent, invokes it with
// a math question, and streams the response deltas to stdout.
package main

import (
	"context"
	"fmt"
	"log"

	agentsdk "github.com/urmzd/graph-agent-dev-kit/agent"
	"github.com/urmzd/graph-agent-dev-kit/agent/core"
	"github.com/urmzd/graph-agent-dev-kit/agent/provider/ollama"
)

func main() {
	// Create Ollama client and adapter.
	client := ollama.NewClient("http://localhost:11434", "llama3.2", "")
	adapter := ollama.NewAdapter(client)

	// Define an "add" tool that sums two numbers.
	addTool := &core.ToolFunc{
		Def: core.ToolDef{
			Name:        "add",
			Description: "Add two numbers together",
			Parameters: core.ParameterSchema{
				Type:     "object",
				Required: []string{"a", "b"},
				Properties: map[string]core.PropertyDef{
					"a": {Type: "number", Description: "First number"},
					"b": {Type: "number", Description: "Second number"},
				},
			},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return fmt.Sprintf("%g", a+b), nil
		},
	}

	// Build the agent.
	agent := agentsdk.NewAgent(agentsdk.AgentConfig{
		Name:         "calculator",
		SystemPrompt: "You are a helpful calculator. Use the add tool to perform addition.",
		Provider:     adapter,
		Tools:        core.NewToolRegistry(addTool),
	})

	// Invoke with a user message.
	stream := agent.Invoke(context.Background(), []core.Message{
		core.NewUserMessage("What is 2 + 3?"),
	})

	// Stream deltas and print text content.
	for delta := range stream.Deltas() {
		switch d := delta.(type) {
		case core.TextContentDelta:
			fmt.Print(d.Content)
		case core.ErrorDelta:
			log.Fatal(d.Error)
		case core.DoneDelta:
			fmt.Println()
		}
	}

	if err := stream.Wait(); err != nil {
		log.Fatal(err)
	}
}
