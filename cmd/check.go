package cmd

import (
	"github.com/Seicrypto/torcontroller/initializer"
	"github.com/spf13/cobra"
)

var fixFlag bool // Default value will be false unless explicitly set by the user.

// CheckCmd is the main command for checking and optionally fixing issues.
var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check system environment for Torcontroller compatibility",
	Long:  `Checks the system for required services and configurations, such as Tor, Privoxy, iptables, and IPv6 support.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Read the fix flag from the command-line input.
		fixFlag, _ := cmd.Flags().GetBool("fix")

		// Perform the environment check based on the flag.
		initializer.CheckEnvironment(fixFlag)
	},
}
