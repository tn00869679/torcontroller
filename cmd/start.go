package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func waitForSocketReady(socketPath string, timeout time.Duration) error {
	start := time.Now()

	for {
		if _, err := os.Stat(socketPath); err == nil {
			// Check if the listener has been activated.
			conn, err := net.Dial("unix", socketPath)
			if err == nil {
				conn.Close()
				return nil
			}
		}

		// Check for timeout
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for socket %s to be ready", socketPath)
		}

		// Wait 100ms and retry
		time.Sleep(100 * time.Millisecond)
	}
}

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a Torcontroller listener",
	Args:  cobra.NoArgs,
	// RunE, not Run: the exit code is the only part of this a script can act
	// on. Reporting success while the daemon refused to install any rules --
	// which is what the port and torrc checks do when something is wrong --
	// would let automation carry on believing its traffic goes through Tor.
	RunE: func(cmd *cobra.Command, args []string) error {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to locate the torcontroller binary: %w", err)
		}

		command := exec.Command(execPath, "start-background")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		if err := command.Start(); err != nil {
			return fmt.Errorf("failed to start the background listener: %w", err)
		}

		err = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", command.Process.Pid)), 0644)
		if err != nil {
			fmt.Printf("Error writing PID file: %v\n", err)
		}

		fmt.Printf("Torcontroller listener started with PID: %d\n", command.Process.Pid)

		// Wait for the socket file to be generated and make sure the listener is started.
		if err := waitForSocketReady(socketPath, 10*time.Second); err != nil {
			return fmt.Errorf("the background listener never became ready: %w", err)
		}

		// Use SocketInteractionHandler to send commands
		handler := &SocketInteractionHandler{
			Adapter: &UnixSocketAdapter{SocketPath: socketPath},
		}

		response, err := handler.SendCommandAndGetResponse("start")
		if err != nil {
			return fmt.Errorf("failed to reach the listener: %w", err)
		}

		// The daemon answers with its own error text rather than closing the
		// connection, so a reply is not by itself a success. This is the path
		// taken when the port or torrc checks refuse to install any rules.
		if strings.HasPrefix(strings.TrimSpace(response), "Error:") {
			return fmt.Errorf("%s", strings.TrimSpace(response))
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Response: %s\n", response)
		return nil
	},
}
