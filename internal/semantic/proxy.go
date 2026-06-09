package semantic

import (
	"strings"

	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/model"
)

// ProxyAnalyzer detects Transparent, UUPS, and Beacon proxy contracts.
type ProxyAnalyzer struct{}

func (a *ProxyAnalyzer) Name() string { return "proxy" }

var proxyBasePatterns = []string{
	"transparentupgradeableproxy",
	"uupsupgradeable",
	"beaconproxy",
	"upgradeablebeacon",
	"erc1967proxy",
	"proxyadmin",
}

var proxyNamePatterns = []string{
	"proxy",
	"upgradeable",
}

func (a *ProxyAnalyzer) Analyze(g *graph.Graph) error {
	for _, node := range g.Nodes {
		if isProxy(node, g) {
			node.Metadata["ComponentType"] = string(model.ComponentTypeProxy)
			node.Metadata["Upgradeable"] = "true"
			node.Metadata["ProxyPattern"] = detectProxyPattern(node, g)
		}
	}
	return nil
}

func isProxy(node *graph.Node, g *graph.Graph) bool {
	lower := strings.ToLower(node.ID)
	for _, pat := range proxyNamePatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	for _, base := range inheritanceList(node, g) {
		lower := strings.ToLower(base)
		for _, pat := range proxyBasePatterns {
			if strings.Contains(lower, pat) {
				return true
			}
		}
	}
	return false
}

func detectProxyPattern(node *graph.Node, g *graph.Graph) string {
	inherits := inheritanceList(node, g)
	for _, base := range inherits {
		lower := strings.ToLower(base)
		switch {
		case strings.Contains(lower, "uups"):
			return "UUPS"
		case strings.Contains(lower, "transparent"):
			return "Transparent"
		case strings.Contains(lower, "beacon"):
			return "Beacon"
		}
	}
	return "Unknown"
}
