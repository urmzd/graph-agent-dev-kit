// Package main demonstrates a parent agent delegating work to a child
// "researcher" sub-agent. The researcher has a mock "search" tool. The parent
// agent routes tasks to the researcher via the automatic delegate_to_ tool,
// and child deltas are forwarded with ToolExecDelta attribution.
package main

import (
	"context"
	"fmt"
	"log"

	agentsdk "github.com/urmzd/saige/agent"
	"github.com/urmzd/saige/agent/core"
	"github.com/urmzd/saige/agent/provider/ollama"
)

func main() {
	// Shared provider for both parent and child.
	client := ollama.NewClient("http://localhost:11434", "llama3.2", "")
	adapter := ollama.NewAdapter(client)

	// Mock search tool for the researcher sub-agent.
	searchTool := &core.ToolFunc{
		Def: core.ToolDef{
			Name:        "search",
			Description: "Search the web for information on a topic",
			Parameters: core.ParameterSchema{
				Type:     "object",
				Required: []string{"query"},
				Properties: map[string]core.PropertyDef{
					"query": {Type: "string", Description: "Search query"},
				},
			},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			query, _ := args["query"].(string)
			return fmt.Sprintf("Results for %q: Go 1.24 adds generic type aliases, "+
				"improved range-over-func iterators, and Swiss Tables map implementation.", query), nil
		},
	}

	// Build the parent agent with a researcher sub-agent.
	agent := agentsdk.NewAgent(agentsdk.AgentConfig{
		Name:         "coordinator",
		SystemPrompt: "You coordinate research tasks. Delegate research to the researcher.",
		Provider:     adapter,
		SubAgents: []agentsdk.SubAgentDef{
			{
				Name:         "researcher",
				Description:  "A research specialist that can search for information",
				SystemPrompt: "You are a research assistant. Use the search tool to find information.",
				Provider:     adapter,
				Tools:        core.NewToolRegistry(searchTool),
			},
		},
	})

	// Invoke with a research request.
	stream := agent.Invoke(context.Background(), []core.Message{
		core.NewUserMessage("Research the latest Go features"),
	})

	// Consume deltas, showing sub-agent attribution.
	for delta := range stream.Deltas() {
		switch d := delta.(type) {
		case core.TextContentDelta:
			fmt.Print(d.Content)
		case core.ToolExecStartDelta:
			fmt.Printf("\n[tool-start] %s (id=%s)\n", d.Name, d.ToolCallID)
		case core.ToolExecDelta:
			if inner, ok := d.Inner.(core.TextContentDelta); ok {
				fmt.Printf("  [sub-agent %s] %s", d.ToolCallID, inner.Content)
			}
		case core.ToolExecEndDelta:
			fmt.Printf("\n[tool-end] id=%s\n", d.ToolCallID)
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
