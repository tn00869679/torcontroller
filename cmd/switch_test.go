package cmd_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/Seicrypto/torcontroller/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSwitchCmd(t *testing.T) {
	// Create a mock response map
	responseMap := map[string]string{
		"switch": "Switch successful: New IP is 192.0.2.1\n",
	}

	mockSocket := &MockSocket{
		ResponseMap: responseMap,
		CloseSignal: make(chan struct{}),
	}
	mockHandler := &cmd.SocketInteractionHandler{
		Adapter: mockSocket,
	}

	rootCmd := &cobra.Command{}
	rootCmd.SetContext(context.WithValue(context.Background(), cmd.HandlerKey, mockHandler))
	rootCmd.AddCommand(cmd.SwitchCmd)

	// Execute the traffic command
	args := []string{"switch"}
	rootCmd.SetArgs(args)

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Switch successful: New IP is 192.0.2.1")
}
