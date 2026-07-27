package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
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
	Run: func(cmd *cobra.Command, args []string) {
		execPath, err := os.Executable()
		if err != nil {
			fmt.Println("Error getting executable path:", err)
			return
		}

		command := exec.Command(execPath, "start-background")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		err = command.Start()
		if err != nil {
			fmt.Println("Error starting background process:", err)
			return
		}

		err = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", command.Process.Pid)), 0644)
		if err != nil {
			fmt.Printf("Error writing PID file: %v\n", err)
		}

		fmt.Printf("Torcontroller listener started with PID: %d\n", command.Process.Pid)

		// Wait for the socket file to be generated and make sure the listener is started.
		if err := waitForSocketReady(socketPath, 10*time.Second); err != nil {
			fmt.Printf("Error waiting for socket: %v\n", err)
			return
		}

		// Use SocketInteractionHandler to send commands
		handler := &SocketInteractionHandler{
			Adapter: &UnixSocketAdapter{SocketPath: socketPath},
		}

		response, err := handler.SendCommandAndGetResponse("start")
		if err != nil {
			fmt.Printf("Error sending command: %v\n", err)
			return
		}

		fmt.Printf("Response: %s\n", response)
	},
}
