package core

import (
	"context"
	"fmt"
	"sync"
)

// ToolDef describes a tool's schema for the LLM.
type ToolDef struct {
	Name        string
	Description string
	Parameters  ParameterSchema
}

// ParameterSchema is a JSON-Schema-like definition for tool parameters.
type ParameterSchema struct {
	Type       string
	Required   []string
	Properties map[string]PropertyDef
}

// PropertyDef describes a single parameter property using JSON Schema fields.
type PropertyDef struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Items       *PropertyDef           `json:"items,omitempty"`
	Properties  map[string]PropertyDef `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Default     any                    `json:"default,omitempty"`
}

// Tool is the base interface all tools implement.
type Tool interface {
	Definition() ToolDef
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// ToolFunc adapts a plain function into a Tool.
type ToolFunc struct {
	Def ToolDef
	Fn  func(ctx context.Context, args map[string]any) (string, error)
}

func (t *ToolFunc) Definition() ToolDef {
	return t.Def
}

func (t *ToolFunc) Execute(ctx context.Context, args map[string]any) (string, error) {
	return t.Fn(ctx, args)
}

// ToolRegistry holds named tools. It is safe for concurrent use.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewToolRegistry creates a registry from the given tools.
func NewToolRegistry(tools ...Tool) *ToolRegistry {
	r := &ToolRegistry{tools: make(map[string]Tool, len(tools))}
	for _, t := range tools {
		r.tools[t.Definition().Name] = t
	}
	return r
}

// Get returns a tool by name.
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Definition().Name] = t
}

// Definitions returns all tool definitions.
func (r *ToolRegistry) Definitions() []ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

// Execute runs a tool by name.
func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}
	return t.Execute(ctx, args)
}
