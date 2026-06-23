package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/smartbom/smartbom/internal/discovery"
	"github.com/smartbom/smartbom/internal/git"
	"github.com/smartbom/smartbom/internal/graph"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Export the dependency graph as Graphviz DOT format",
	Long: `Discover and parse smart contracts from a repository, then write the
dependency graph to stdout (or a file) in Graphviz DOT format.

Example:
  smartbom graph --repo ./local/path
  smartbom graph --repo https://github.com/org/protocol --output graph.dot`,
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().StringP("repo", "r", "", "Repository URL or local path (required)")
	graphCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	graphCmd.Flags().StringP("branch", "b", "", "Branch or tag to check out (default: HEAD)")
	graphCmd.Flags().StringP("workdir", "w", "", "Working directory for clones (default: system temp)")
	graphCmd.Flags().BoolP("keep", "k", false, "Keep the cloned repository after export")
	_ = graphCmd.MarkFlagRequired("repo")
}

func runGraph(cmd *cobra.Command, _ []string) error {
	repoURL, _ := cmd.Flags().GetString("repo")
	outputPath, _ := cmd.Flags().GetString("output")
	branch, _ := cmd.Flags().GetString("branch")
	workDir, _ := cmd.Flags().GetString("workdir")
	keepRepo, _ := cmd.Flags().GetBool("keep")

	// ── 1. Clone / locate repository ─────────────────────────────────────────
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

	// ── 2. Discover ───────────────────────────────────────────────────────────
	sc := discovery.NewFileScanner()
	project, err := sc.Scan(repoPath)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}

	if len(project.SolidityFiles) == 0 && len(project.VyperFiles) == 0 {
		return fmt.Errorf("no smart contract source files found in %s", repoPath)
	}

	// ── 3. Parse ──────────────────────────────────────────────────────────────
	parsedFiles, err := parseAll(project)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// ── 4. Build graph ────────────────────────────────────────────────────────
	g := graph.NewBuilder().Build(parsedFiles)

	// ── 5. Emit DOT ───────────────────────────────────────────────────────────
	var w io.Writer = os.Stdout
	if outputPath != "" {
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("open output: %w", err)
		}
		defer f.Close()
		w = f
	}

	writeDOT(g, w)
	return nil
}

// writeDOT serialises g as a Graphviz DOT digraph.
// Each node carries its type as a label attribute; each edge carries its
// relationship as a label.
func writeDOT(g *graph.Graph, w io.Writer) {
	fmt.Fprintln(w, "digraph smartbom {")
	fmt.Fprintln(w, `  rankdir=LR;`)
	fmt.Fprintln(w, `  node [shape=box fontname="Helvetica"];`)

	for id, node := range g.Nodes {
		label := fmt.Sprintf("%s\\n[%s]", id, node.Type)
		fmt.Fprintf(w, "  %q [label=%q];\n", id, label)
	}

	for _, e := range g.Edges {
		fmt.Fprintf(w, "  %q -> %q [label=%q];\n", e.From, e.To, e.Relationship)
	}

	fmt.Fprintln(w, "}")
}
