package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var TrafficCmd = &cobra.Command{
	Use:   "traffic",
	Short: "Check the current traffic metrics for the Tor service",
	Args:  cobra.NoArgs,
	// RunE, not Run: a monitoring script reading these figures needs to be
	// able to tell "no traffic" apart from "the daemon could not be reached".
	RunE: func(cmd *cobra.Command, args []string) error {
		handler, ok := cmd.Context().Value(HandlerKey).(*SocketInteractionHandler)
		if !ok || handler == nil {
			return fmt.Errorf("socket handler not initialised")
		}

		response, err := handler.SendCommandAndGetResponse("traffic")
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
