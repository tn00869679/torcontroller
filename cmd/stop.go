package cmd

import (
	"fmt"
	"strconv"

	"github.com/Seicrypto/torcontroller/internal/controller"
	"github.com/Seicrypto/torcontroller/internal/singleton/logger"
	"github.com/spf13/cobra"
)

var StopCmd = &cobra.Command{
	Use:   "stop [socketPath]",
	Short: "Stop a Torcontroller listener",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the cmd ctx
		handler, ok := cmd.Context().Value(HandlerKey).(*SocketInteractionHandler)
		if !ok || handler == nil {
			fmt.Println("Error: handler not initialized")
			return
		}

		fs, ok := cmd.Context().Value(FileSystem).(controller.FileSystem)
		if !ok {
			fmt.Println("Error: FileSystem not initialized")
			return
		}

		logger, ok := cmd.Context().Value(Logger).(*logger.Logger)
		if !ok {
			fmt.Println("Error: Logger not initialized")
			return
		}

		response, err := handler.SendCommandAndGetResponse("stop")
		if err != nil {
			logger.Error(fmt.Sprintf("Error sending command: %v", err))
			fmt.Printf("Error sending command: %v\n", err)
			return
		}

		if response != "Done\n" {
			logger.Warn(fmt.Sprintf("Unexpected response from server: %s", response))
			fmt.Printf("Unexpected response from server: %s\n", response)
			return
		}

		logger.Info("Server confirmed successful stop.")
		fmt.Println("Server confirmed successful stop.")

		// Read the PID file
		data, err := fs.ReadFile(pidFile)
		if err != nil {
			fmt.Printf("Error reading PID file: %v\n", err)
			return
		}

		pid, err := strconv.Atoi(string(data))
		if err != nil {
			fmt.Printf("Error parsing PID: %v\n", err)
			return
		}

		proc, err := fs.FindProcess(pid)
		if err != nil {
			fmt.Printf("Error finding process: %v\n", err)
			return
		}

		err = proc.Kill()
		if err != nil {
			fmt.Printf("Error killing process: %v\n", err)
			return
		}

		fs.Remove(socketPath)
		fs.Remove(pidFile)
		logger.Info(fmt.Sprintf("Torcontroller listener at %s stopped successfully.\n", socketPath))
		fmt.Fprintf(cmd.OutOrStdout(), "Torcontroller listener at %s stopped successfully.\n", socketPath)
	},
}
