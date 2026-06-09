package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	cdxenc "github.com/CycloneDX/cyclonedx-go"
	"github.com/spf13/cobra"

	"github.com/smartbom/smartbom/internal/cyclonedx"
	"github.com/smartbom/smartbom/internal/discovery"
	"github.com/smartbom/smartbom/internal/git"
	"github.com/smartbom/smartbom/internal/graph"
	"github.com/smartbom/smartbom/internal/parser"
	"github.com/smartbom/smartbom/internal/parser/solidity"
	"github.com/smartbom/smartbom/internal/semantic"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan a repository and generate a CycloneDX BOM",
	Long: `Clone a GitHub (or local) repository, discover smart contract source files,
build a dependency graph, run semantic analysis, and emit a CycloneDX JSON BOM.

Example:
  smartbom scan --repo https://github.com/org/defi-protocol
  smartbom scan --repo ./local/path --output output/bom.json`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringP("repo", "r", "", "Repository URL or local path (required)")
	scanCmd.Flags().StringP("output", "o", "output/bom.json", "Output file path")
	scanCmd.Flags().StringP("branch", "b", "", "Branch or tag to check out (default: HEAD)")
	scanCmd.Flags().StringP("workdir", "w", "", "Working directory for clones (default: system temp)")
	scanCmd.Flags().BoolP("keep", "k", false, "Keep the cloned repository after scanning")
	_ = scanCmd.MarkFlagRequired("repo")
}

func runScan(cmd *cobra.Command, _ []string) error {
	repoURL, _ := cmd.Flags().GetString("repo")
	outputPath, _ := cmd.Flags().GetString("output")
	branch, _ := cmd.Flags().GetString("branch")
	workDir, _ := cmd.Flags().GetString("workdir")
	keepRepo, _ := cmd.Flags().GetBool("keep")

	// ── 1. Clone ──────────────────────────────────────────────────────────────
	var repoPath string
	var mgr *git.Manager

	if isLocalPath(repoURL) {
		repoPath = repoURL
		slog.Info("using local repository", "path", repoPath)
	} else {
		mgr = git.NewManager(workDir)
		slog.Info("cloning repository", "url", repoURL)
		repo, err := mgr.CloneWithRef(repoURL, branch)
		if err != nil {
			return fmt.Errorf("clone: %w", err)
		}
		repoPath = repo.Path
		if !keepRepo {
			defer func() {
				if cleanErr := mgr.Cleanup(repo); cleanErr != nil {
					slog.Warn("cleanup failed", "err", cleanErr)
				}
			}()
		}
	}
	fmt.Println("Repository ready:", repoPath)

	// ── 2. Discovery ──────────────────────────────────────────────────────────
	slog.Info("scanning for source files")
	scanner := discovery.NewFileScanner()
	project, err := scanner.Scan(repoPath)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	printDiscovery(project)

	if len(project.SolidityFiles) == 0 && len(project.VyperFiles) == 0 {
		return fmt.Errorf("no smart contract source files found in %s", repoPath)
	}

	// ── 3. Parse ──────────────────────────────────────────────────────────────
	slog.Info("parsing source files")
	parsedFiles, err := parseAll(project)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	totalContracts := 0
	for _, pf := range parsedFiles {
		totalContracts += len(pf.Contracts)
	}
	fmt.Printf("Contracts parsed: %d (from %d files)\n", totalContracts, len(parsedFiles))

	// ── 4. Build dependency graph ──────────────────────────────────────────────
	slog.Info("building dependency graph")
	builder := graph.NewBuilder()
	g := builder.Build(parsedFiles)

	nodes, edges := g.Stats()
	fmt.Printf("Graph: %d nodes, %d edges\n", nodes, edges)

	// ── 5. Semantic analysis ───────────────────────────────────────────────────
	slog.Info("running semantic analysis")
	pipeline := semantic.DefaultPipeline()
	if err := pipeline.Run(g); err != nil {
		return fmt.Errorf("semantic analysis: %w", err)
	}
	printSemanticSummary(g)

	// ── 6. Generate CycloneDX BOM ─────────────────────────────────────────────
	slog.Info("generating CycloneDX BOM")
	bomBuilder := cyclonedx.NewBuilder()
	bom, err := bomBuilder.Build(g)
	if err != nil {
		return fmt.Errorf("BOM generation: %w", err)
	}

	// ── 7. Write output ────────────────────────────────────────────────────────
	if err := writeOutput(bom, outputPath); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	fmt.Printf("\nCycloneDX BOM saved: %s\n", outputPath)
	return nil
}

// parseAll runs the appropriate parser for each discovered file type.
func parseAll(project *discovery.Project) ([]*parser.ParsedFile, error) {
	solParser := solidity.New()
	var files []*parser.ParsedFile

	for _, path := range project.SolidityFiles {
		slog.Debug("parsing", "file", path)
		pf, err := solParser.Parse(path)
		if err != nil {
			slog.Warn("parse error (skipping)", "file", path, "err", err)
			continue
		}
		files = append(files, pf)
	}

	// Vyper, Rust, Move: stubs — extend here when parsers are available.
	for _, path := range project.VyperFiles {
		slog.Debug("vyper parser not yet implemented, skipping", "file", path)
	}

	return files, nil
}

func writeOutput(bom interface{}, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Use the CycloneDX encoder for spec-compliant JSON output.
	enc := cdxenc.NewBOMEncoder(f, cdxenc.BOMFileFormatJSON)
	enc.SetPretty(true)

	if cdxBOM, ok := bom.(*cdxenc.BOM); ok {
		return enc.Encode(cdxBOM)
	}

	// Fallback: plain JSON.
	return json.NewEncoder(f).Encode(bom)
}

func isLocalPath(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] == '/' || s[0] == '.' || s[0] == '~'
}

func printDiscovery(p *discovery.Project) {
	fmt.Printf("\nDiscovery Results:\n")
	fmt.Printf("  Solidity files : %d\n", len(p.SolidityFiles))
	fmt.Printf("  Vyper files    : %d\n", len(p.VyperFiles))
	fmt.Printf("  Rust (Cargo)   : %d\n", len(p.RustFiles))
	fmt.Printf("  Move           : %d\n", len(p.MoveFiles))
	fmt.Printf("  Config files   : %d\n", len(p.ConfigFiles))
}

func printSemanticSummary(g *graph.Graph) {
	counts := map[string]int{}
	for _, n := range g.Nodes {
		if ct, ok := n.Metadata["ComponentType"].(string); ok {
			counts[ct]++
		}
	}
	fmt.Printf("\nSemantic Analysis:\n")
	for _, label := range []string{"Token", "Proxy", "Oracle", "Governance", "Treasury"} {
		if c := counts[label]; c > 0 {
			fmt.Printf("  %-12s: %d\n", label, c)
		}
	}
}
