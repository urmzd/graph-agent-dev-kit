// Package main demonstrates file upload with content negotiation. It registers
// a file:// resolver that reads files from disk, attaches a FileContent block
// to a user message, and lets the agent's file pipeline resolve the URI and
// check the provider's ContentNegotiator for native media type support.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	agentsdk "github.com/urmzd/graph-agent-dev-kit/agent"
	"github.com/urmzd/graph-agent-dev-kit/agent/core"
	"github.com/urmzd/graph-agent-dev-kit/agent/provider/ollama"
)

func main() {
	client := ollama.NewClient("http://localhost:11434", "llava", "")
	adapter := ollama.NewAdapter(client)

	// Check what content types the provider supports natively.
	support := adapter.ContentSupport()
	fmt.Println("Provider native types:")
	for mt, ok := range support.NativeTypes {
		if ok {
			fmt.Printf("  - %s\n", mt)
		}
	}

	// file:// resolver that reads from the local filesystem.
	fileResolver := core.ResolverFunc(func(ctx context.Context, uri string) (core.ResolvedFile, error) {
		path := strings.TrimPrefix(uri, "file://")
		data, err := os.ReadFile(path)
		if err != nil {
			return core.ResolvedFile{}, fmt.Errorf("read file %s: %w", path, err)
		}
		// Detect media type from content.
		mediaType := core.MediaType(http.DetectContentType(data))
		return core.ResolvedFile{Data: data, MediaType: mediaType}, nil
	})

	// Build agent with the file resolver.
	agent := agentsdk.NewAgent(agentsdk.AgentConfig{
		Name:         "multimodal-agent",
		SystemPrompt: "You are a helpful assistant that can analyze images and files.",
		Provider:     adapter,
		Resolvers: map[string]core.Resolver{
			"file": fileResolver,
		},
	})

	// Build a message with text and a file attachment.
	imagePath := "example.png"
	if len(os.Args) > 1 {
		imagePath = os.Args[1]
	}

	msg := core.NewUserMessageWithFiles(
		"Describe what you see in this image.",
		core.FileContent{
			URI:      "file://" + imagePath,
			Filename: imagePath,
		},
	)

	// Invoke the agent.
	stream := agent.Invoke(context.Background(), []core.Message{msg})

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
