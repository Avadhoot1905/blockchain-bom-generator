package graph

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/smartbom/smartbom/internal/parser"
)

// Builder constructs a Graph from a slice of ParsedFiles.
// It resolves imports to known contracts where possible, and creates
// external-dependency nodes for third-party packages (e.g. OpenZeppelin).
type Builder struct{}

// NewBuilder returns a Builder.
func NewBuilder() *Builder { return &Builder{} }

// Build ingests all parsed files and returns the populated graph.
func (b *Builder) Build(files []*parser.ParsedFile) *Graph {
	g := New()

	// First pass: register all local contract nodes so that cross-file
	// inheritance and import resolution can find them.
	contractByName := make(map[string]*parser.Contract)
	for _, pf := range files {
		for i := range pf.Contracts {
			c := &pf.Contracts[i]
			// Merge file-level imports with contract-level imports so that
			// all contracts in a file see the shared imports.
			allImports := mergeImports(pf.FileImports, c.Imports)
			node := &Node{
				ID:   c.Name,
				Type: c.Kind,
				Metadata: map[string]any{
					"SourceFile": c.SourceFile,
					"Inherits":   c.Inherits,
					"Imports":    allImports,
					"Functions":  functionNames(c.Functions),
					"Events":     c.Events,
					"Modifiers":  c.Modifiers,
				},
			}
			g.UpsertNode(node)
			contractByName[c.Name] = c
		}
	}

	// Second pass: build edges.
	for _, pf := range files {
		for i := range pf.Contracts {
			c := &pf.Contracts[i]

			// Inheritance edges.
			for _, base := range c.Inherits {
				if _, ok := g.Nodes[base]; !ok {
					// External base — create a stub node.
					g.UpsertNode(&Node{
						ID:   base,
						Type: "external",
						Metadata: map[string]any{
							"PackageName": inferPackage(base),
						},
					})
				}
				if err := g.AddEdge(Edge{
					From:         c.Name,
					To:           base,
					Relationship: "inherits",
				}); err != nil {
					slog.Debug("edge error", "err", err)
				}
			}

			// Import edges — use merged (file-level + contract-level) imports.
			allImports := mergeImports(pf.FileImports, c.Imports)
			for _, imp := range allImports {
				targetID := resolveImport(imp, pf.Path, contractByName)
				if targetID == "" {
					continue
				}
				if _, ok := g.Nodes[targetID]; !ok {
					g.UpsertNode(&Node{
						ID:   targetID,
						Type: "external",
						Metadata: map[string]any{
							"ImportPath":  imp,
							"PackageName": inferPackage(imp),
						},
					})
				}
				if err := g.AddEdge(Edge{
					From:         c.Name,
					To:           targetID,
					Relationship: "imports",
				}); err != nil {
					slog.Debug("edge error", "err", err)
				}
			}
		}
	}

	return g
}

// resolveImport maps an import path to a graph node ID.
// Local imports (./foo.sol) are resolved to their filename stem.
// External imports (@openzeppelin/...) are resolved to their package name.
func resolveImport(imp, sourceFile string, known map[string]*parser.Contract) string {
	// Relative local import.
	if strings.HasPrefix(imp, "./") || strings.HasPrefix(imp, "../") {
		stem := strings.TrimSuffix(filepath.Base(imp), ".sol")
		if _, ok := known[stem]; ok {
			return stem
		}
		return stem // still create a node for it
	}
	// External package import — derive a canonical package-level ID.
	return packageNodeID(imp)
}

// packageNodeID creates a stable node ID for an external import path.
// "@openzeppelin/contracts/token/ERC20/ERC20.sol" → "@openzeppelin/contracts"
func packageNodeID(imp string) string {
	if strings.HasPrefix(imp, "@") {
		parts := strings.SplitN(imp, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	// Fallback: use the directory portion.
	dir := filepath.Dir(imp)
	if dir == "." || dir == "" {
		return imp
	}
	return dir
}

// inferPackage guesses the human-readable package name from a contract name
// or import path.
func inferPackage(s string) string {
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "openzeppelin"):
		return "OpenZeppelin"
	case strings.Contains(lower, "chainlink"):
		return "Chainlink"
	case strings.Contains(lower, "uniswap"):
		return "Uniswap"
	case strings.Contains(lower, "aave"):
		return "Aave"
	}
	return s
}

func functionNames(fns []parser.Function) []string {
	names := make([]string, len(fns))
	for i, f := range fns {
		names[i] = f.Name
	}
	return names
}

// mergeImports combines file-level and contract-level imports, deduplicating.
func mergeImports(fileImports, contractImports []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, imp := range fileImports {
		if !seen[imp] {
			seen[imp] = true
			result = append(result, imp)
		}
	}
	for _, imp := range contractImports {
		if !seen[imp] {
			seen[imp] = true
			result = append(result, imp)
		}
	}
	return result
}
