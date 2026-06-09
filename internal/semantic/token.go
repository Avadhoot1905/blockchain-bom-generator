package semantic

import (
	"strings"

	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/model"
)

// TokenAnalyzer detects ERC20 / ERC721 / ERC1155 token contracts.
type TokenAnalyzer struct{}

func (a *TokenAnalyzer) Name() string { return "token" }

func (a *TokenAnalyzer) Analyze(g *graph.Graph) error {
	for _, node := range g.Nodes {
		if isToken(node, g) {
			node.Metadata["ComponentType"] = string(model.ComponentTypeToken)
			node.Metadata["TokenStandard"] = detectTokenStandard(node, g)
		}
	}
	return nil
}

func isToken(node *graph.Node, g *graph.Graph) bool {
	inherits := inheritanceList(node, g)
	for _, base := range inherits {
		upper := strings.ToUpper(base)
		if strings.Contains(upper, "ERC20") ||
			strings.Contains(upper, "ERC721") ||
			strings.Contains(upper, "ERC1155") {
			return true
		}
	}
	name := strings.ToUpper(node.ID)
	return strings.Contains(name, "TOKEN") || strings.Contains(name, "NFT")
}

func detectTokenStandard(node *graph.Node, g *graph.Graph) string {
	inherits := inheritanceList(node, g)
	for _, base := range inherits {
		upper := strings.ToUpper(base)
		switch {
		case strings.Contains(upper, "ERC1155"):
			return string(model.TokenStandardERC1155)
		case strings.Contains(upper, "ERC721"):
			return string(model.TokenStandardERC721)
		case strings.Contains(upper, "ERC20"):
			return string(model.TokenStandardERC20)
		}
	}
	return ""
}
