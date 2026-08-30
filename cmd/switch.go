package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var SwitchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch the Tor IP",
	Args:  cobra.NoArgs,
	// RunE, not Run: someone switching circuits is usually trying to shed an
	// address that has been blocked or seen. Reporting success when the
	// circuit did not change tells them they have a new identity when they
	// still have the old one.
	RunE: func(cmd *cobra.Command, args []string) error {
		handler, ok := cmd.Context().Value(HandlerKey).(*SocketInteractionHandler)
		if !ok || handler == nil {
			return fmt.Errorf("socket handler not initialised")
		}

		response, err := handler.SendCommandAndGetResponse("switch")
		if err != nil {
			return fmt.Errorf("failed to reach the listener: %w", err)
		}

		// The daemon replies with its own error text rather than closing the
		// connection, so a reply is not by itself a success.
		if strings.HasPrefix(strings.TrimSpace(response), "Error:") {
			return fmt.Errorf("%s", strings.TrimSpace(response))
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Response: %s\n", response)
		return nil
	},
}
