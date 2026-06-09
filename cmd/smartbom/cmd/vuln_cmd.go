package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var vulnCmd = &cobra.Command{
	Use:   "vuln",
	Short: "Run vulnerability scanners against a BOM or repository",
	Long:  `Execute the vulnerability scanning engine. (Framework present; scanners not yet implemented.)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("'vuln' command not yet implemented — vulnerability scanner framework is wired but scanners are stubs")
	},
}
