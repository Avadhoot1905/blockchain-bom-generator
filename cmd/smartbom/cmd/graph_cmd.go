package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Visualize the dependency graph of a scanned repository",
	Long:  `Export or visualize the internal dependency graph. (Not yet implemented.)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("'graph' command not yet implemented — use 'scan' to generate a BOM")
	},
}
