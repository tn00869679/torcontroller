package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var TrafficCmd = &cobra.Command{
	Use:   "traffic",
	Short: "Check the current traffic metrics for the Tor service",
	Run: func(cmd *cobra.Command, args []string) {
		handler, ok := cmd.Context().Value(HandlerKey).(*SocketInteractionHandler)
		if !ok || handler == nil {
			fmt.Println("Error: handler not initialized")
			return
		}

		response, err := handler.SendCommandAndGetResponse("traffic")
		if err != nil {
			fmt.Printf("Error executing command: %v\n", err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Response: %s\n", response)
	},
}
