package semantic

import (
	"testing"

	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/model"
)

func makeNode(id, nodeType string, inherits []string) *graph.Node {
	n := &graph.Node{
		ID:   id,
		Type: nodeType,
		Metadata: map[string]any{
			"Inherits": inherits,
		},
	}
	return n
}

func buildGraph(nodes ...*graph.Node) *graph.Graph {
	g := graph.New()
	for _, n := range nodes {
		g.UpsertNode(n)
	}
	return g
}

func componentType(node *graph.Node) string {
	if v, ok := node.Metadata["ComponentType"].(string); ok {
		return v
	}
	return ""
}

// --- Token -------------------------------------------------------------------

func TestTokenAnalyzer_ERC20(t *testing.T) {
	node := makeNode("MyToken", "contract", []string{"ERC20", "Ownable"})
	g := buildGraph(node)

	a := &TokenAnalyzer{}
	if err := a.Analyze(g); err != nil {
		t.Fatal(err)
	}
	if componentType(node) != string(model.ComponentTypeToken) {
		t.Errorf("expected Token, got %q", componentType(node))
	}
	if node.Metadata["TokenStandard"] != string(model.TokenStandardERC20) {
		t.Errorf("expected ERC20 standard, got %v", node.Metadata["TokenStandard"])
	}
}

func TestTokenAnalyzer_ERC721(t *testing.T) {
	node := makeNode("MyNFT", "contract", []string{"ERC721", "Ownable"})
	g := buildGraph(node)
	_ = (&TokenAnalyzer{}).Analyze(g)
	if node.Metadata["TokenStandard"] != string(model.TokenStandardERC721) {
		t.Errorf("expected ERC721, got %v", node.Metadata["TokenStandard"])
	}
}

func TestTokenAnalyzer_NoMatch(t *testing.T) {
	node := makeNode("Vault", "contract", []string{"Ownable"})
	g := buildGraph(node)
	_ = (&TokenAnalyzer{}).Analyze(g)
	if componentType(node) == string(model.ComponentTypeToken) {
		t.Error("Vault should not be classified as Token")
	}
}

// --- Proxy -------------------------------------------------------------------

func TestProxyAnalyzer_UUPS(t *testing.T) {
	node := makeNode("UpgradeableToken", "contract", []string{"ERC20Upgradeable", "UUPSUpgradeable", "OwnableUpgradeable"})
	g := buildGraph(node)
	_ = (&ProxyAnalyzer{}).Analyze(g)
	if componentType(node) != string(model.ComponentTypeProxy) {
		t.Errorf("expected Proxy, got %q", componentType(node))
	}
	if node.Metadata["ProxyPattern"] != "UUPS" {
		t.Errorf("expected UUPS pattern, got %v", node.Metadata["ProxyPattern"])
	}
}

func TestProxyAnalyzer_ByName(t *testing.T) {
	node := makeNode("TransparentProxy", "contract", []string{"Ownable"})
	g := buildGraph(node)
	_ = (&ProxyAnalyzer{}).Analyze(g)
	if componentType(node) != string(model.ComponentTypeProxy) {
		t.Errorf("expected Proxy by name, got %q", componentType(node))
	}
}

// --- Oracle ------------------------------------------------------------------

func TestOracleAnalyzer_Chainlink(t *testing.T) {
	node := makeNode("PriceOracle", "contract", []string{"AggregatorV3Interface"})
	node.Metadata["Imports"] = []string{"@chainlink/contracts/src/v0.8/interfaces/AggregatorV3Interface.sol"}
	g := buildGraph(node)
	_ = (&OracleAnalyzer{}).Analyze(g)
	if componentType(node) != string(model.ComponentTypeOracle) {
		t.Errorf("expected Oracle, got %q", componentType(node))
	}
}

// --- Governance --------------------------------------------------------------

func TestGovernanceAnalyzer_Governor(t *testing.T) {
	node := makeNode("MyGovernor", "contract", []string{
		"Governor", "GovernorVotes", "GovernorTimelockControl",
	})
	g := buildGraph(node)
	_ = (&GovernanceAnalyzer{}).Analyze(g)
	if componentType(node) != string(model.ComponentTypeGovernance) {
		t.Errorf("expected Governance, got %q", componentType(node))
	}
}

func TestGovernanceAnalyzer_ByName(t *testing.T) {
	node := makeNode("ProtocolDAO", "contract", []string{})
	g := buildGraph(node)
	_ = (&GovernanceAnalyzer{}).Analyze(g)
	if componentType(node) != string(model.ComponentTypeGovernance) {
		t.Errorf("expected Governance by name, got %q", componentType(node))
	}
}

// --- Treasury ----------------------------------------------------------------

func TestTreasuryAnalyzer_Vault(t *testing.T) {
	node := makeNode("ProtocolVault", "contract", []string{"Ownable"})
	g := buildGraph(node)
	_ = (&TreasuryAnalyzer{}).Analyze(g)
	if componentType(node) != string(model.ComponentTypeTreasury) {
		t.Errorf("expected Treasury, got %q", componentType(node))
	}
}

// --- Pipeline ----------------------------------------------------------------

func TestPipeline(t *testing.T) {
	token := makeNode("MyToken", "contract", []string{"ERC20"})
	proxy := makeNode("MyProxy", "contract", []string{"UUPSUpgradeable"})
	other := makeNode("Registry", "contract", []string{})

	g := buildGraph(token, proxy, other)

	p := DefaultPipeline()
	if err := p.Run(g); err != nil {
		t.Fatal(err)
	}

	if componentType(token) != string(model.ComponentTypeToken) {
		t.Errorf("token type = %q", componentType(token))
	}
	if componentType(proxy) != string(model.ComponentTypeProxy) {
		t.Errorf("proxy type = %q", componentType(proxy))
	}
}
