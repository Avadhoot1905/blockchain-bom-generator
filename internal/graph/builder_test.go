package graph

import (
	"testing"

	"github.com/smartbom/smartbom/internal/parser"
)

func TestBuilderBasic(t *testing.T) {
	files := []*parser.ParsedFile{
		{
			Path:     "/repo/contracts/Token.sol",
			Language: "solidity",
			Contracts: []parser.Contract{
				{
					Name:       "MyToken",
					Kind:       "contract",
					SourceFile: "/repo/contracts/Token.sol",
					Inherits:   []string{"ERC20", "Ownable"},
					Imports:    []string{"@openzeppelin/contracts/token/ERC20/ERC20.sol"},
					Functions: []parser.Function{
						{Name: "mint", Visibility: "public"},
					},
				},
			},
		},
	}

	b := NewBuilder()
	g := b.Build(files)

	if _, ok := g.Nodes["MyToken"]; !ok {
		t.Fatal("MyToken node not found")
	}
	// ERC20 and Ownable should be created as external stubs.
	if _, ok := g.Nodes["ERC20"]; !ok {
		t.Error("ERC20 stub not found")
	}
	if _, ok := g.Nodes["Ownable"]; !ok {
		t.Error("Ownable stub not found")
	}
	// Inherits edges.
	found := false
	for _, e := range g.Edges {
		if e.From == "MyToken" && e.To == "ERC20" && e.Relationship == "inherits" {
			found = true
		}
	}
	if !found {
		t.Error("expected inherits edge MyToken → ERC20")
	}
}

func TestBuilderMultiFile(t *testing.T) {
	files := []*parser.ParsedFile{
		{
			Path: "/repo/contracts/IERC20.sol",
			Contracts: []parser.Contract{
				{Name: "IERC20", Kind: "interface", SourceFile: "/repo/contracts/IERC20.sol"},
			},
		},
		{
			Path: "/repo/contracts/Token.sol",
			Contracts: []parser.Contract{
				{
					Name:       "Token",
					Kind:       "contract",
					SourceFile: "/repo/contracts/Token.sol",
					Inherits:   []string{"IERC20"},
					Imports:    []string{"./IERC20.sol"},
				},
			},
		},
	}

	b := NewBuilder()
	g := b.Build(files)

	if _, ok := g.Nodes["IERC20"]; !ok {
		t.Fatal("IERC20 not registered")
	}
	// Local import resolved to IERC20 — check import edge.
	found := false
	for _, e := range g.Edges {
		if e.From == "Token" && e.Relationship == "imports" {
			found = true
		}
	}
	if !found {
		t.Error("expected imports edge from Token")
	}
}

func TestPackageNodeID(t *testing.T) {
	cases := []struct {
		imp  string
		want string
	}{
		{"@openzeppelin/contracts/token/ERC20/ERC20.sol", "@openzeppelin/contracts"},
		{"@chainlink/contracts/src/v0.8/interfaces/AggregatorV3Interface.sol", "@chainlink/contracts"},
		{"hardhat/console.sol", "hardhat"},
	}
	for _, c := range cases {
		got := packageNodeID(c.imp)
		if got != c.want {
			t.Errorf("packageNodeID(%q) = %q, want %q", c.imp, got, c.want)
		}
	}
}
