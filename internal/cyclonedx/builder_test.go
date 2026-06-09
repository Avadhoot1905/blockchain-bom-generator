package cyclonedx

import (
	"strings"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/model"
)

func makeTestGraph() *graph.Graph {
	g := graph.New()
	g.UpsertNode(&graph.Node{
		ID:   "MyToken",
		Type: "contract",
		Metadata: map[string]any{
			"ComponentType": string(model.ComponentTypeToken),
			"TokenStandard": string(model.TokenStandardERC20),
			"Inherits":      []string{"ERC20", "Ownable"},
			"SourceFile":    "/repo/contracts/MyToken.sol",
		},
	})
	g.UpsertNode(&graph.Node{
		ID:   "ERC20",
		Type: "external",
		Metadata: map[string]any{
			"PackageName": "OpenZeppelin",
		},
	})
	g.UpsertNode(&graph.Node{
		ID:   "MyProxy",
		Type: "contract",
		Metadata: map[string]any{
			"ComponentType": string(model.ComponentTypeProxy),
			"Upgradeable":   "true",
			"ProxyPattern":  "UUPS",
		},
	})
	_ = g.AddEdge(graph.Edge{From: "MyToken", To: "ERC20", Relationship: "inherits"})
	return g
}

func TestBuildBOM(t *testing.T) {
	b := NewBuilder()
	bom, err := b.Build(makeTestGraph())
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	if bom.SpecVersion != cdx.SpecVersion1_6 {
		t.Errorf("spec version = %v, want 1.6", bom.SpecVersion)
	}
	if bom.Components == nil || len(*bom.Components) == 0 {
		t.Fatal("expected components")
	}
	if bom.Dependencies == nil || len(*bom.Dependencies) == 0 {
		t.Fatal("expected dependencies")
	}
	if bom.Metadata == nil {
		t.Fatal("expected metadata")
	}
	if !strings.HasPrefix(bom.SerialNumber, "urn:uuid:") {
		t.Errorf("unexpected serial number: %s", bom.SerialNumber)
	}
}

func TestBOMComponentProperties(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeTestGraph())

	comp := findComponent(*bom.Components, "MyToken")
	if comp == nil {
		t.Fatal("MyToken component not found")
	}

	props := *comp.Properties
	if !hasProperty(props, "smartbom:ContractType", "Token") {
		t.Errorf("expected ContractType=Token in properties, got %v", props)
	}
	if !hasProperty(props, "smartbom:TokenStandard", "ERC20") {
		t.Errorf("expected TokenStandard=ERC20, got %v", props)
	}
}

func TestBOMProxyProperties(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeTestGraph())

	comp := findComponent(*bom.Components, "MyProxy")
	if comp == nil {
		t.Fatal("MyProxy component not found")
	}

	props := *comp.Properties
	if !hasProperty(props, "smartbom:Upgradeable", "true") {
		t.Errorf("expected Upgradeable=true")
	}
	if !hasProperty(props, "smartbom:ProxyPattern", "UUPS") {
		t.Errorf("expected ProxyPattern=UUPS")
	}
}

func TestSanitizeBOMRef(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"MyToken", "MyToken"},
		{"@openzeppelin/contracts", "-openzeppelin-contracts"},
		{"My Token", "My-Token"},
	}
	for _, c := range cases {
		got := sanitizeBOMRef(c.input)
		if got != c.want {
			t.Errorf("sanitizeBOMRef(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func findComponent(components []cdx.Component, name string) *cdx.Component {
	for i := range components {
		if components[i].Name == name {
			return &components[i]
		}
	}
	return nil
}

func hasProperty(props []cdx.Property, name, value string) bool {
	for _, p := range props {
		if p.Name == name && p.Value == value {
			return true
		}
	}
	return false
}
