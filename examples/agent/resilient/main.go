// Package main demonstrates composing retry and fallback providers for
// resilient LLM calls. A primary adapter is wrapped with retry logic,
// then combined with a secondary adapter via fallback. If the primary
// fails after retries, the fallback adapter is tried automatically.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	agentsdk "github.com/urmzd/graph-agent-dev-kit/agent"
	"github.com/urmzd/graph-agent-dev-kit/agent/core"
	"github.com/urmzd/graph-agent-dev-kit/agent/provider/fallback"
	"github.com/urmzd/graph-agent-dev-kit/agent/provider/ollama"
	"github.com/urmzd/graph-agent-dev-kit/agent/provider/retry"
)

func main() {
	// Primary provider: llama3.2 with retry.
	primaryClient := ollama.NewClient("http://localhost:11434", "llama3.2", "")
	primaryAdapter := ollama.NewAdapter(primaryClient)

	retryProvider := retry.New(primaryAdapter, retry.Config{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
	})

	// Secondary provider: different model as fallback.
	secondaryClient := ollama.NewClient("http://localhost:11434", "mistral", "")
	secondaryAdapter := ollama.NewAdapter(secondaryClient)

	// Compose: retry the primary, then fall back to the secondary.
	composed := fallback.New(retryProvider, secondaryAdapter)

	// Build agent with the composed provider.
	agent := agentsdk.NewAgent(agentsdk.AgentConfig{
		Name:         "resilient-agent",
		SystemPrompt: "You are a helpful assistant.",
		Provider:     composed,
	})

	// Invoke and stream the response.
	stream := agent.Invoke(context.Background(), []core.Message{
		core.NewUserMessage("Explain the benefits of retry and fallback patterns in distributed systems."),
	})

	for delta := range stream.Deltas() {
		switch d := delta.(type) {
		case core.TextContentDelta:
			fmt.Print(d.Content)
		case core.UsageDelta:
			fmt.Printf("\n[usage] prompt=%d completion=%d latency=%s\n",
				d.PromptTokens, d.CompletionTokens, d.Latency)
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
