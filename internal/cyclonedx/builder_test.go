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

// ── SBOM foundation: license, hash, version, crypto properties ───────────────

func makeEnrichedGraph() *graph.Graph {
	g := graph.New()
	g.UpsertNode(&graph.Node{
		ID:   "PermitToken",
		Type: "contract",
		Metadata: map[string]any{
			"License":          "MIT",
			"SolidityVersion":  "^0.8.20",
			"SourceHash":       "abc123def456abc123def456abc123def456abc123def456abc123def456abc12345",
			"CryptoPrimitives": []string{"ECDSA", "EIP712"},
			"CryptoCategories": []string{"DigitalSignature", "TypedDataSigning"},
		},
	})
	g.UpsertNode(&graph.Node{
		ID:   "@openzeppelin/contracts",
		Type: "external",
		Metadata: map[string]any{
			"Version":     "^4.9.0",
			"PackageName": "OpenZeppelin",
		},
	})
	return g
}

func TestBOMComponentLicense(t *testing.T) {
	b := NewBuilder()
	bom, err := b.Build(makeEnrichedGraph())
	if err != nil {
		t.Fatal(err)
	}
	comp := findComponent(*bom.Components, "PermitToken")
	if comp == nil {
		t.Fatal("PermitToken not found")
	}
	if comp.Licenses == nil || len(*comp.Licenses) == 0 {
		t.Fatal("expected Licenses to be set")
	}
	if (*comp.Licenses)[0].License == nil || (*comp.Licenses)[0].License.ID != "MIT" {
		t.Errorf("expected SPDX license ID MIT, got %+v", (*comp.Licenses)[0])
	}
}

func TestBOMComponentHash(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeEnrichedGraph())
	comp := findComponent(*bom.Components, "PermitToken")
	if comp == nil {
		t.Fatal("PermitToken not found")
	}
	if comp.Hashes == nil || len(*comp.Hashes) == 0 {
		t.Fatal("expected Hashes to be set")
	}
	h := (*comp.Hashes)[0]
	if h.Algorithm != "SHA-256" {
		t.Errorf("hash algorithm = %q, want SHA-256", h.Algorithm)
	}
	if h.Value == "" {
		t.Error("hash value should not be empty")
	}
}

func TestBOMComponentVersion_FromSolidityPragma(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeEnrichedGraph())
	comp := findComponent(*bom.Components, "PermitToken")
	if comp == nil {
		t.Fatal("PermitToken not found")
	}
	if comp.Version != "^0.8.20" {
		t.Errorf("Version = %q, want ^0.8.20", comp.Version)
	}
}

func TestBOMComponentVersion_FromPackageJSON(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeEnrichedGraph())
	comp := findComponent(*bom.Components, "@openzeppelin/contracts")
	if comp == nil {
		t.Fatal("@openzeppelin/contracts not found")
	}
	if comp.Version != "^4.9.0" {
		t.Errorf("external package Version = %q, want ^4.9.0", comp.Version)
	}
}

func TestBOMCryptoPrimitivesProperties(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeEnrichedGraph())
	comp := findComponent(*bom.Components, "PermitToken")
	if comp == nil || comp.Properties == nil {
		t.Fatal("PermitToken or its properties not found")
	}
	props := *comp.Properties
	if !hasProperty(props, "smartbom:CryptoPrimitives", "ECDSA") {
		t.Error("expected smartbom:CryptoPrimitives=ECDSA")
	}
	if !hasProperty(props, "smartbom:CryptoPrimitives", "EIP712") {
		t.Error("expected smartbom:CryptoPrimitives=EIP712")
	}
	if !hasProperty(props, "smartbom:CryptoCategories", "DigitalSignature") {
		t.Error("expected smartbom:CryptoCategories=DigitalSignature")
	}
	if !hasProperty(props, "smartbom:CryptoCategories", "TypedDataSigning") {
		t.Error("expected smartbom:CryptoCategories=TypedDataSigning")
	}
}

func TestBOMSolidityVersionProperty(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeEnrichedGraph())
	comp := findComponent(*bom.Components, "PermitToken")
	if comp == nil || comp.Properties == nil {
		t.Fatal("PermitToken or its properties not found")
	}
	if !hasProperty(*comp.Properties, "smartbom:SolidityVersion", "^0.8.20") {
		t.Error("expected smartbom:SolidityVersion=^0.8.20 property")
	}
}

func TestBOMLicenseProperty(t *testing.T) {
	b := NewBuilder()
	bom, _ := b.Build(makeEnrichedGraph())
	comp := findComponent(*bom.Components, "PermitToken")
	if comp == nil || comp.Properties == nil {
		t.Fatal("PermitToken or its properties not found")
	}
	if !hasProperty(*comp.Properties, "smartbom:License", "MIT") {
		t.Error("expected smartbom:License=MIT property")
	}
}

func TestBOMBackwardCompatibility(t *testing.T) {
	// Nodes without the new fields should still produce valid BOM output.
	b := NewBuilder()
	bom, err := b.Build(makeTestGraph())
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	if bom.Components == nil || len(*bom.Components) == 0 {
		t.Error("expected components from legacy test graph")
	}
	// Existing properties must still be present.
	comp := findComponent(*bom.Components, "MyToken")
	if comp == nil {
		t.Fatal("MyToken not found")
	}
	if !hasProperty(*comp.Properties, "smartbom:ContractType", "Token") {
		t.Error("backward compat: expected smartbom:ContractType=Token")
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
