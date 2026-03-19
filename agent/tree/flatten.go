package tree

import (
	"fmt"

	"github.com/urmzd/graph-agent-dev-kit/agent/core"
)

// AnnotatedMessage pairs a message with its tree metadata.
type AnnotatedMessage struct {
	Message  core.Message
	NodeID   core.NodeID
	Path     core.TreePath
	Depth    int
	BranchID core.BranchID
	State    core.NodeState
}

// Flatten walks the path from root to the given node and collects messages.
// Archived nodes are skipped. Compacted nodes contribute their summary message.
func (t *Tree) Flatten(toNodeID core.NodeID) ([]core.Message, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.flattenUnlocked(toNodeID)
}

func (t *Tree) flattenUnlocked(toNodeID core.NodeID) ([]core.Message, error) {
	path, err := t.pathUnlocked(toNodeID)
	if err != nil {
		return nil, err
	}

	messages := make([]core.Message, 0, len(path))
	for _, nid := range path {
		node := t.nodes[nid]
		if node.State != core.NodeArchived {
			messages = append(messages, node.Message)
		}
	}
	return messages, nil
}

// FlattenBranch flattens the path from root to the tip of the given branch.
func (t *Tree) FlattenBranch(branch core.BranchID) ([]core.Message, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tipID, ok := t.branches[branch]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrBranchNotFound, branch)
	}
	return t.flattenUnlocked(tipID)
}

// FlattenAnnotated walks the path from root to the given node and returns
// annotated messages with full metadata. Archived nodes are skipped.
// Compacted nodes are included with State: NodeCompacted.
func (t *Tree) FlattenAnnotated(toNodeID core.NodeID) ([]AnnotatedMessage, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.flattenAnnotatedUnlocked(toNodeID)
}

func (t *Tree) flattenAnnotatedUnlocked(toNodeID core.NodeID) ([]AnnotatedMessage, error) {
	path, err := t.pathUnlocked(toNodeID)
	if err != nil {
		return nil, err
	}

	result := make([]AnnotatedMessage, 0, len(path))
	for _, nid := range path {
		node := t.nodes[nid]
		if node.State == core.NodeArchived {
			continue
		}
		tp, err := t.nodePathUnlocked(nid)
		if err != nil {
			return nil, err
		}
		result = append(result, AnnotatedMessage{
			Message:  node.Message,
			NodeID:   node.ID,
			Path:     tp,
			Depth:    node.Depth,
			BranchID: node.BranchID,
			State:    node.State,
		})
	}
	return result, nil
}

// FlattenBranchAnnotated flattens the path from root to the tip of the given
// branch, returning annotated messages.
func (t *Tree) FlattenBranchAnnotated(branch core.BranchID) ([]AnnotatedMessage, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tipID, ok := t.branches[branch]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrBranchNotFound, branch)
	}
	return t.flattenAnnotatedUnlocked(tipID)
}
