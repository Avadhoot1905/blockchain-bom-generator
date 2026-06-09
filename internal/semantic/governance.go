package semantic

import (
	"strings"

	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/model"
)

// GovernanceAnalyzer detects Governor, DAO, and access-control contracts.
type GovernanceAnalyzer struct{}

func (a *GovernanceAnalyzer) Name() string { return "governance" }

var governanceBasePatterns = []string{
	"governor",
	"governorbravo",
	"governorvotes",
	"timelockcontroller",
	"accesscontrol",
	"accesscontrolenumerable",
}

var governanceNamePatterns = []string{
	"governor",
	"governance",
	"dao",
	"timelock",
}

func (a *GovernanceAnalyzer) Analyze(g *graph.Graph) error {
	for _, node := range g.Nodes {
		if isGovernance(node, g) {
			node.Metadata["ComponentType"] = string(model.ComponentTypeGovernance)
		}
	}
	return nil
}

func isGovernance(node *graph.Node, g *graph.Graph) bool {
	lower := strings.ToLower(node.ID)
	for _, pat := range governanceNamePatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	for _, base := range inheritanceList(node, g) {
		lower := strings.ToLower(base)
		for _, pat := range governanceBasePatterns {
			if strings.Contains(lower, pat) {
				return true
			}
		}
	}
	return false
}
