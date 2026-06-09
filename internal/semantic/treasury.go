package semantic

import (
	"strings"

	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/model"
)

// TreasuryAnalyzer detects vault, treasury, and asset custody contracts.
type TreasuryAnalyzer struct{}

func (a *TreasuryAnalyzer) Name() string { return "treasury" }

var treasuryNamePatterns = []string{
	"treasury",
	"vault",
	"safe",
	"multisig",
	"escrow",
	"custody",
	"reserve",
}

var treasuryBasePatterns = []string{
	"gnosis",
	"multisigwallet",
	"safevault",
}

func (a *TreasuryAnalyzer) Analyze(g *graph.Graph) error {
	for _, node := range g.Nodes {
		if isTreasury(node, g) {
			node.Metadata["ComponentType"] = string(model.ComponentTypeTreasury)
		}
	}
	return nil
}

func isTreasury(node *graph.Node, g *graph.Graph) bool {
	lower := strings.ToLower(node.ID)
	for _, pat := range treasuryNamePatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	for _, base := range inheritanceList(node, g) {
		lower := strings.ToLower(base)
		for _, pat := range treasuryBasePatterns {
			if strings.Contains(lower, pat) {
				return true
			}
		}
	}
	return false
}
