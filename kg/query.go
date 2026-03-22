package kg

import "github.com/urmzd/saige/kg/kgtypes"

// FactsToStrings converts facts to string representations.
func FactsToStrings(facts []kgtypes.Fact) []string {
	result := make([]string, len(facts))
	for i, f := range facts {
		result[i] = f.SourceNode.Name + " -> " + f.TargetNode.Name + ": " + f.FactText
	}
	return result
}

// FilterFacts filters facts by a predicate.
func FilterFacts(facts []kgtypes.Fact, pred func(kgtypes.Fact) bool) []kgtypes.Fact {
	result := make([]kgtypes.Fact, 0)
	for _, f := range facts {
		if pred(f) {
			result = append(result, f)
		}
	}
	return result
}

// FilterByType returns facts with a matching relation type.
func FilterByType(facts []kgtypes.Fact, relType string) []kgtypes.Fact {
	return FilterFacts(facts, func(f kgtypes.Fact) bool {
		return f.Name == relType
	})
}

// Subgraph collects a subgraph around a starting node.
func Subgraph(detail *kgtypes.NodeDetail) *kgtypes.GraphData {
	nodes := make([]kgtypes.GraphNode, 0, len(detail.Neighbors)+1)
	nodes = append(nodes, detail.Node)
	nodes = append(nodes, detail.Neighbors...)
	return &kgtypes.GraphData{Nodes: nodes, Edges: detail.Edges}
}
