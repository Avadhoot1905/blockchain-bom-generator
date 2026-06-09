package semantic

import (
	"github.com/smartbom/smartbom/internal/graph"
)

// inheritanceList returns the list of base contracts stored in node metadata.
func inheritanceList(node *graph.Node, g *graph.Graph) []string {
	// First check direct metadata from the parser.
	if inherits, ok := node.Metadata["Inherits"].([]string); ok {
		return inherits
	}
	// Fall back to graph edges with relationship "inherits".
	var result []string
	for _, e := range g.Edges {
		if e.From == node.ID && e.Relationship == "inherits" {
			result = append(result, e.To)
		}
	}
	return result
}

// functionList returns function names stored in node metadata.
func functionList(node *graph.Node) []string {
	if fns, ok := node.Metadata["Functions"].([]string); ok {
		return fns
	}
	return nil
}

// importList returns the imports stored in node metadata.
func importList(node *graph.Node) []string {
	if imports, ok := node.Metadata["Imports"].([]string); ok {
		return imports
	}
	return nil
}
