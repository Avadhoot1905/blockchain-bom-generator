package graph

import (
	"fmt"
)

// Node represents a single vertex in the dependency graph.
// Metadata holds arbitrary key-value pairs so downstream consumers
// (semantic analyzers, BOM builders) can annotate nodes without
// requiring schema changes to this package.
type Node struct {
	ID       string
	Type     string // "contract", "interface", "library", "external", …
	Metadata map[string]any
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From         string
	To           string
	Relationship string // "imports", "inherits", "uses", …
}

// Graph is an in-memory directed dependency graph.
// It is the single source of truth for all downstream processing.
type Graph struct {
	Nodes map[string]*Node
	Edges []Edge
}

// New returns an empty Graph.
func New() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
	}
}

// AddNode inserts a node; returns an error if the ID already exists.
func (g *Graph) AddNode(n *Node) error {
	if _, exists := g.Nodes[n.ID]; exists {
		return fmt.Errorf("node %q already exists", n.ID)
	}
	if n.Metadata == nil {
		n.Metadata = make(map[string]any)
	}
	g.Nodes[n.ID] = n
	return nil
}

// UpsertNode inserts a node or replaces it if the ID already exists.
func (g *Graph) UpsertNode(n *Node) {
	if n.Metadata == nil {
		n.Metadata = make(map[string]any)
	}
	g.Nodes[n.ID] = n
}

// AddEdge appends an edge; both endpoints must exist.
func (g *Graph) AddEdge(e Edge) error {
	if _, ok := g.Nodes[e.From]; !ok {
		return fmt.Errorf("source node %q not found", e.From)
	}
	if _, ok := g.Nodes[e.To]; !ok {
		return fmt.Errorf("target node %q not found", e.To)
	}
	g.Edges = append(g.Edges, e)
	return nil
}

// FindNode returns the node with the given ID, or nil if absent.
func (g *Graph) FindNode(id string) (*Node, bool) {
	n, ok := g.Nodes[id]
	return n, ok
}

// DependenciesOf returns the nodes that nodeID directly depends on
// (i.e. nodes reachable via outgoing edges from nodeID).
func (g *Graph) DependenciesOf(nodeID string) []*Node {
	seen := make(map[string]bool)
	var result []*Node
	for _, e := range g.Edges {
		if e.From == nodeID && !seen[e.To] {
			if n, ok := g.Nodes[e.To]; ok {
				result = append(result, n)
				seen[e.To] = true
			}
		}
	}
	return result
}

// DependentsOf returns the nodes that directly depend on nodeID
// (i.e. nodes with incoming edges from nodeID).
func (g *Graph) DependentsOf(nodeID string) []*Node {
	seen := make(map[string]bool)
	var result []*Node
	for _, e := range g.Edges {
		if e.To == nodeID && !seen[e.From] {
			if n, ok := g.Nodes[e.From]; ok {
				result = append(result, n)
				seen[e.From] = true
			}
		}
	}
	return result
}

// TopologicalSort returns nodes in dependency order (sources first).
// Returns an error if the graph contains a cycle.
func (g *Graph) TopologicalSort() ([]*Node, error) {
	inDegree := make(map[string]int, len(g.Nodes))
	for id := range g.Nodes {
		inDegree[id] = 0
	}
	for _, e := range g.Edges {
		inDegree[e.To]++
	}

	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []*Node
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, g.Nodes[id])
		for _, e := range g.Edges {
			if e.From == id {
				inDegree[e.To]--
				if inDegree[e.To] == 0 {
					queue = append(queue, e.To)
				}
			}
		}
	}

	if len(sorted) != len(g.Nodes) {
		return nil, fmt.Errorf("dependency cycle detected in graph")
	}
	return sorted, nil
}

// NodesByType returns all nodes matching the given type string.
func (g *Graph) NodesByType(nodeType string) []*Node {
	var result []*Node
	for _, n := range g.Nodes {
		if n.Type == nodeType {
			result = append(result, n)
		}
	}
	return result
}

// EdgesFrom returns all edges originating from the given node ID.
func (g *Graph) EdgesFrom(nodeID string) []Edge {
	var result []Edge
	for _, e := range g.Edges {
		if e.From == nodeID {
			result = append(result, e)
		}
	}
	return result
}

// Stats returns a summary of graph size for logging.
func (g *Graph) Stats() (nodes int, edges int) {
	return len(g.Nodes), len(g.Edges)
}
