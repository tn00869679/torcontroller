package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show TorController version",
	Long:  "TorController CLI version 1.1.0\n\nVisit https://github.com/Seicrypto/torcontroller for release notes and updates.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), "TorController version 1.1.0")
	},
}
