package types

import "context"

// Marker is a routing annotation attached to a tool. When a marked tool is
// invoked, the agent loop pauses, emits a MarkerDelta to the consumer, and
// waits for resolution before proceeding.
type Marker struct {
	Kind    string         // e.g. "human_approval", "audit", "rate_limit"
	Message string         // human-readable description of what's being gated
	Meta    map[string]any // arbitrary metadata for the consumer
}

// MarkedTool wraps a Tool with one or more Markers.
type MarkedTool struct {
	Inner   Tool
	Markers []Marker
}

func (m *MarkedTool) Definition() ToolDef { return m.Inner.Definition() }

func (m *MarkedTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	return m.Inner.Execute(ctx, args)
}

// WithMarkers wraps a tool with markers.
func WithMarkers(tool Tool, markers ...Marker) *MarkedTool {
	return &MarkedTool{Inner: tool, Markers: markers}
}
