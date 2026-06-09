package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "smartbom",
	Short: "SmartBOM — Blockchain SBOM/CBOM Generator",
	Long: `SmartBOM generates hybrid Software/Component Bills of Materials (SBOM/CBOM)
for blockchain and smart contract repositories.

It clones the target repository, discovers Solidity/Vyper/Rust/Move source
files, builds a dependency graph, runs semantic analysis, and emits a
CycloneDX 1.6 JSON BOM.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := slog.LevelInfo
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})))
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable debug logging")
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(vulnCmd)
}
