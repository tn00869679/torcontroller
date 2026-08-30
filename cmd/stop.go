package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tn00869679/torcontroller/internal/controller"
	"github.com/tn00869679/torcontroller/internal/singleton/logger"
)

var StopCmd = &cobra.Command{
	Use:   "stop [socketPath]",
	Short: "Stop a Torcontroller listener",
	Args:  cobra.NoArgs,
	// RunE, not Run: leaving redirection rules in place while reporting
	// success is the worst outcome this command has. A script that treats a
	// zero exit as "the machine is back to normal" would carry on with every
	// connection still going through Tor -- or worse, through a chain whose
	// daemon is gone.
	RunE: func(cmd *cobra.Command, args []string) error {
		handler, ok := cmd.Context().Value(HandlerKey).(*SocketInteractionHandler)
		if !ok || handler == nil {
			return fmt.Errorf("socket handler not initialised")
		}

		fs, ok := cmd.Context().Value(FileSystem).(controller.FileSystem)
		if !ok {
			return fmt.Errorf("file system not initialised")
		}

		log, ok := cmd.Context().Value(Logger).(*logger.Logger)
		if !ok {
			return fmt.Errorf("logger not initialised")
		}

		response, err := handler.SendCommandAndGetResponse("stop")
		if err != nil {
			log.Error(fmt.Sprintf("Error sending command: %v", err))
			return fmt.Errorf("failed to reach the listener: %w", err)
		}

		if strings.TrimSpace(response) != "Done" {
			log.Warn(fmt.Sprintf("Unexpected response from server: %s", response))
			return fmt.Errorf("the listener did not confirm the stop, and rules may still be installed: %s", strings.TrimSpace(response))
		}

		log.Info("Server confirmed successful stop.")
		fmt.Fprintln(cmd.OutOrStdout(), "Server confirmed successful stop.")

		// The rules are already gone by this point; what follows is tidying up
		// the listener itself. Failing here leaves a stray process rather than
		// a redirected network, so it is reported but described as such.
		data, err := fs.ReadFile(pidFile)
		if err != nil {
			return fmt.Errorf("rules were removed but the listener PID could not be read from %s: %w", pidFile, err)
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return fmt.Errorf("rules were removed but %s does not contain a PID: %w", pidFile, err)
		}

		proc, err := fs.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("rules were removed but listener process %d was not found: %w", pid, err)
		}

		if err := proc.Kill(); err != nil {
			return fmt.Errorf("rules were removed but listener process %d could not be stopped: %w", pid, err)
		}

		fs.Remove(socketPath)
		fs.Remove(pidFile)
		log.Info(fmt.Sprintf("Torcontroller listener at %s stopped successfully.", socketPath))
		fmt.Fprintf(cmd.OutOrStdout(), "Torcontroller listener at %s stopped successfully.\n", socketPath)
		return nil
	},
}
