// Package main demonstrates concurrent sub-agent execution. A parent
// "coordinator" agent delegates to two specialist sub-agents — a researcher
// and a fact-checker — in parallel. When the LLM returns multiple
// delegate_to_ tool calls in one response, the SDK executes them
// concurrently via goroutines.
package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	agentsdk "github.com/urmzd/graph-agent-dev-kit/agent"
	"github.com/urmzd/graph-agent-dev-kit/agent/core"
	"github.com/urmzd/graph-agent-dev-kit/agent/provider/ollama"
)

func main() {
	client := ollama.NewClient("http://localhost:11434", "llama3.2", "")
	adapter := ollama.NewAdapter(client)

	// Track concurrent execution with an atomic counter.
	var running int32

	// Mock search tool for the researcher.
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
			n := atomic.AddInt32(&running, 1)
			fmt.Printf("  [search] started (concurrent tasks: %d)\n", n)
			time.Sleep(500 * time.Millisecond) // simulate latency
			atomic.AddInt32(&running, -1)
			return fmt.Sprintf("Results for %q: Go 1.24 adds generic type aliases, "+
				"improved range-over-func iterators, and Swiss Tables map implementation.", query), nil
		},
	}

	// Mock verify tool for the fact-checker.
	verifyTool := &core.ToolFunc{
		Def: core.ToolDef{
			Name:        "verify",
			Description: "Verify a factual claim",
			Parameters: core.ParameterSchema{
				Type:     "object",
				Required: []string{"claim"},
				Properties: map[string]core.PropertyDef{
					"claim": {Type: "string", Description: "The claim to verify"},
				},
			},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			claim, _ := args["claim"].(string)
			n := atomic.AddInt32(&running, 1)
			fmt.Printf("  [verify] started (concurrent tasks: %d)\n", n)
			time.Sleep(500 * time.Millisecond) // simulate latency
			atomic.AddInt32(&running, -1)
			return fmt.Sprintf("Verified: %q is accurate according to official Go release notes.", claim), nil
		},
	}

	agent := agentsdk.NewAgent(agentsdk.AgentConfig{
		Name: "coordinator",
		SystemPrompt: `You coordinate research and fact-checking tasks.
You have two specialists available:
- researcher: searches for information
- fact_checker: verifies factual claims

IMPORTANT: Always delegate to BOTH specialists at the same time by calling
delegate_to_researcher AND delegate_to_fact_checker in the same response.
This allows them to work concurrently.`,
		Provider: adapter,
		SubAgents: []agentsdk.SubAgentDef{
			{
				Name:         "researcher",
				Description:  "A research specialist that searches for information on topics",
				SystemPrompt: "You are a research assistant. Use the search tool to find information. Be concise.",
				Provider:     adapter,
				Tools:        core.NewToolRegistry(searchTool),
			},
			{
				Name:         "fact_checker",
				Description:  "A fact-checking specialist that verifies claims for accuracy",
				SystemPrompt: "You are a fact-checker. Use the verify tool to check claims. Be concise.",
				Provider:     adapter,
				Tools:        core.NewToolRegistry(verifyTool),
			},
		},
	})

	stream := agent.Invoke(context.Background(), []core.Message{
		core.NewUserMessage("Research the latest Go 1.24 features and verify that Go 1.24 introduced generic type aliases."),
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
