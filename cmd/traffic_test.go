package cmd_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/Seicrypto/torcontroller/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestTrafficCmd(t *testing.T) {
	// Create a mock response map
	responseMap := map[string]string{
		"traffic": "Traffic metrics: 123 connections, 45MB data\n",
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
	rootCmd.AddCommand(cmd.TrafficCmd)

	// Execute the traffic command
	args := []string{"traffic"}
	rootCmd.SetArgs(args)

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Traffic metrics: 123 connections, 45MB data")
}
