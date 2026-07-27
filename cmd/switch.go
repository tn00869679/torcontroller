package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var SwitchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch the Tor IP",
	Run: func(cmd *cobra.Command, args []string) {
		handler, ok := cmd.Context().Value(HandlerKey).(*SocketInteractionHandler)
		if !ok || handler == nil {
			fmt.Println("Error: handler not initialized")
			return
		}

		response, err := handler.SendCommandAndGetResponse("switch")
		if err != nil {
			fmt.Printf("Error executing command: %v\n", err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Response: %s\n", response)
	},
}
