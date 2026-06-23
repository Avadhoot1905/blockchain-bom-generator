// Package cyclonedx translates the internal dependency graph into a
// CycloneDX 1.6 BOM using the official cyclonedx-go library.
// Blockchain-specific metadata is stored in CycloneDX properties so that
// the standard schema is not modified.
package cyclonedx

import (
	"fmt"
	"log/slog"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
	"github.com/smartbom/smartbom/internal/graph"
)

const (
	toolName    = "SmartBOM"
	toolVersion = "0.1.0"
	specVersion = cdx.SpecVersion1_6
)

// Builder converts a populated graph.Graph into a CycloneDX BOM.
type Builder struct{}

// NewBuilder returns a Builder.
func NewBuilder() *Builder { return &Builder{} }

// Build produces a CycloneDX BOM from the dependency graph.
func (b *Builder) Build(g *graph.Graph) (*cdx.BOM, error) {
	bom := cdx.NewBOM()
	bom.SpecVersion = specVersion
	bom.Version = 1
	bom.SerialNumber = "urn:uuid:" + uuid.NewString()

	bom.Metadata = buildMetadata()

	components, err := buildComponents(g)
	if err != nil {
		return nil, fmt.Errorf("build components: %w", err)
	}
	bom.Components = &components

	deps := buildDependencies(g)
	bom.Dependencies = &deps

	nodes, edges := g.Stats()
	slog.Info("BOM generated",
		"components", len(components),
		"dependencies", len(deps),
		"graph_nodes", nodes,
		"graph_edges", edges,
	)
	return bom, nil
}

func buildMetadata() *cdx.Metadata {
	toolVer := toolVersion
	return &cdx.Metadata{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Tools: &cdx.ToolsChoice{
			Components: &[]cdx.Component{
				{
					Type:    cdx.ComponentTypeApplication,
					Name:    toolName,
					Version: toolVer,
				},
			},
		},
		Properties: &[]cdx.Property{
			{Name: "smartbom:schema", Value: "blockchain-cbom"},
		},
	}
}

func buildComponents(g *graph.Graph) ([]cdx.Component, error) {
	var components []cdx.Component

	for id, node := range g.Nodes {
		// Use explicit package version when available; fall back to Solidity pragma.
		version := stringMeta(node, "Version")
		if version == "" {
			version = stringMeta(node, "SolidityVersion")
		}

		comp := cdx.Component{
			BOMRef:  sanitizeBOMRef(id),
			Name:    id,
			Type:    nodeTypeToComponentType(node.Type),
			Version: version,
		}

		if src := stringMeta(node, "SourceFile"); src != "" {
			comp.Description = "Source: " + src
		}

		// SPDX license.
		if lic := stringMeta(node, "License"); lic != "" {
			licenses := cdx.Licenses{
				cdx.LicenseChoice{License: &cdx.License{ID: lic}},
			}
			comp.Licenses = &licenses
		}

		// Source integrity hash.
		if h := stringMeta(node, "SourceHash"); h != "" {
			hashes := []cdx.Hash{
				{Algorithm: cdx.HashAlgoSHA256, Value: h},
			}
			comp.Hashes = &hashes
		}

		props := buildProperties(node)
		if len(props) > 0 {
			comp.Properties = &props
		}

		components = append(components, comp)
	}
	return components, nil
}

func buildDependencies(g *graph.Graph) []cdx.Dependency {
	// Group edges by source node.
	depMap := make(map[string][]string)
	for _, e := range g.Edges {
		from := sanitizeBOMRef(e.From)
		to := sanitizeBOMRef(e.To)
		depMap[from] = append(depMap[from], to)
	}

	deps := make([]cdx.Dependency, 0, len(depMap))
	for ref, targets := range depMap {
		t := make([]string, len(targets))
		copy(t, targets)
		deps = append(deps, cdx.Dependency{
			Ref:          ref,
			Dependencies: &t,
		})
	}
	return deps
}

func buildProperties(node *graph.Node) []cdx.Property {
	var props []cdx.Property

	add := func(name, value string) {
		if value != "" {
			props = append(props, cdx.Property{Name: name, Value: value})
		}
	}

	add("smartbom:ContractType", stringMeta(node, "ComponentType"))
	add("smartbom:TokenStandard", stringMeta(node, "TokenStandard"))
	add("smartbom:Upgradeable", stringMeta(node, "Upgradeable"))
	add("smartbom:ProxyPattern", stringMeta(node, "ProxyPattern"))
	add("smartbom:OracleProvider", stringMeta(node, "OracleProvider"))
	add("smartbom:PackageName", stringMeta(node, "PackageName"))
	add("smartbom:SourceFile", stringMeta(node, "SourceFile"))
	add("smartbom:SolidityVersion", stringMeta(node, "SolidityVersion"))
	add("smartbom:License", stringMeta(node, "License"))

	// Inheritance list.
	if inherits, ok := node.Metadata["Inherits"].([]string); ok && len(inherits) > 0 {
		for _, base := range inherits {
			props = append(props, cdx.Property{Name: "smartbom:Inherits", Value: base})
		}
	}

	// CBOM: cryptographic primitives (e.g. ["ECDSA", "Keccak256"]).
	if prims, ok := node.Metadata["CryptoPrimitives"].([]string); ok {
		for _, p := range prims {
			props = append(props, cdx.Property{Name: "smartbom:CryptoPrimitives", Value: p})
		}
	}

	// CBOM: cryptographic categories (e.g. ["DigitalSignature", "HashFunction"]).
	if cats, ok := node.Metadata["CryptoCategories"].([]string); ok {
		for _, c := range cats {
			props = append(props, cdx.Property{Name: "smartbom:CryptoCategories", Value: c})
		}
	}

	return props
}

// nodeTypeToComponentType maps graph node types to CycloneDX component types.
func nodeTypeToComponentType(nodeType string) cdx.ComponentType {
	switch nodeType {
	case "contract", "abstract":
		return cdx.ComponentTypeFile
	case "interface":
		return cdx.ComponentTypeFile
	case "library":
		return cdx.ComponentTypeLibrary
	case "external":
		return cdx.ComponentTypeLibrary
	default:
		return cdx.ComponentTypeFile
	}
}

// sanitizeBOMRef replaces characters that are invalid in CycloneDX BOM refs.
func sanitizeBOMRef(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '/' || c == ' ' || c == '@' {
			out[i] = '-'
		} else {
			out[i] = c
		}
	}
	return string(out)
}

func stringMeta(node *graph.Node, key string) string {
	if v, ok := node.Metadata[key].(string); ok {
		return v
	}
	return ""
}
