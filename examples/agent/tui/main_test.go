package main

import (
	"bytes"
	"strings"
	"testing"

	agentsdk "github.com/urmzd/graph-agent-dev-kit/agent"
	"github.com/urmzd/graph-agent-dev-kit/agent/agenttest"
	"github.com/urmzd/graph-agent-dev-kit/agent/core"
	"github.com/urmzd/graph-agent-dev-kit/agent/tui"
)

func TestStreamVerbose(t *testing.T) {
	provider := &agenttest.ScriptedProvider{
		Responses: [][]core.Delta{
			agenttest.ToolCallResponse("tc-1", "delegate_to_researcher", map[string]any{
				"task": "research Go features",
			}),
			agenttest.TextResponse("Here is a summary of Go features."),
		},
	}

	researcherProvider := &agenttest.ScriptedProvider{
		Responses: [][]core.Delta{
			agenttest.TextResponse("Go 1.24 adds generic type aliases."),
		},
	}

	agent := agentsdk.NewAgent(agentsdk.AgentConfig{
		Name:         "coordinator",
		SystemPrompt: "Coordinate research.",
		Provider:     provider,
		SubAgents: []agentsdk.SubAgentDef{
			{
				Name:         "researcher",
				Description:  "Research specialist",
				SystemPrompt: "Research things.",
				Provider:     researcherProvider,
			},
		},
	})

	stream := agent.Invoke(t.Context(), []core.Message{
		core.NewUserMessage("Research Go features"),
	})

	info := agent.Info()
	header := tui.AgentHeader{
		Name:      info.Name,
		Provider:  info.Provider,
		Tools:     info.Tools,
		SubAgents: info.SubAgents,
	}

	var buf bytes.Buffer
	result := tui.StreamVerbose(header, stream.Deltas(), &buf)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	if result.Text == "" {
		t.Fatal("expected non-empty text output")
	}

	output := buf.String()
	t.Logf("Output:\n%s", output)

	if !strings.Contains(output, "coordinator") {
		t.Error("expected agent name in output")
	}

	if !strings.Contains(result.Text, "summary") && !strings.Contains(result.Text, "Go") {
		t.Errorf("unexpected text: %s", result.Text)
	}
}

func TestVerboseFormatters(t *testing.T) {
	start := tui.FormatDelegateStart("researcher")
	if start == "" {
		t.Fatal("FormatDelegateStart returned empty string")
	}

	output := tui.FormatAgentOutput("researcher", "some output")
	if output == "" {
		t.Fatal("FormatAgentOutput returned empty string")
	}

	done := tui.FormatAgentDone("researcher")
	if done == "" {
		t.Fatal("FormatAgentDone returned empty string")
	}

	errMsg := tui.FormatAgentError("researcher", "timeout")
	if errMsg == "" {
		t.Fatal("FormatAgentError returned empty string")
	}
}
