package semantic

import (
	"strings"

	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/model"
)

// OracleAnalyzer detects Chainlink price feeds and other oracle adapters.
type OracleAnalyzer struct{}

func (a *OracleAnalyzer) Name() string { return "oracle" }

var oracleBasePatterns = []string{
	"aggregatorv3interface",
	"aggregatorinterface",
	"feedregistryinterface",
	"ipricefeed",
	"priceoracle",
}

var oracleFunctions = []string{
	"latestrounddata",
	"getprice",
	"getlatestprice",
	"latestanswer",
}

var oracleImportPatterns = []string{
	"chainlink",
	"aggregator",
	"pricefeed",
	"oracle",
}

func (a *OracleAnalyzer) Analyze(g *graph.Graph) error {
	for _, node := range g.Nodes {
		if isOracle(node, g) {
			node.Metadata["ComponentType"] = string(model.ComponentTypeOracle)
			node.Metadata["OracleProvider"] = detectOracleProvider(node)
		}
	}
	return nil
}

func isOracle(node *graph.Node, g *graph.Graph) bool {
	for _, base := range inheritanceList(node, g) {
		lower := strings.ToLower(base)
		for _, pat := range oracleBasePatterns {
			if strings.Contains(lower, pat) {
				return true
			}
		}
	}
	for _, fn := range functionList(node) {
		lower := strings.ToLower(fn)
		for _, pat := range oracleFunctions {
			if lower == pat {
				return true
			}
		}
	}
	for _, imp := range importList(node) {
		lower := strings.ToLower(imp)
		for _, pat := range oracleImportPatterns {
			if strings.Contains(lower, pat) {
				return true
			}
		}
	}
	return false
}

func detectOracleProvider(node *graph.Node) string {
	imports, _ := node.Metadata["Imports"].([]string)
	for _, imp := range imports {
		lower := strings.ToLower(imp)
		if strings.Contains(lower, "chainlink") {
			return "Chainlink"
		}
		if strings.Contains(lower, "uniswap") && strings.Contains(lower, "oracle") {
			return "Uniswap TWAP"
		}
		if strings.Contains(lower, "band") {
			return "Band Protocol"
		}
	}
	return "Unknown"
}
