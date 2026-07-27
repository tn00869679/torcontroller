package cmd_test

import (
	"bytes"
	"testing"

	"github.com/Seicrypto/torcontroller/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestVersionCmd(t *testing.T) {
	// Mock
	rootCmd := &cobra.Command{}
	rootCmd.AddCommand(cmd.VersionCmd)

	// Capture output
	output := new(bytes.Buffer)
	rootCmd.SetOut(output)
	rootCmd.SetArgs([]string{"version"})

	// Execute the command
	err := rootCmd.Execute()
	assert.NoError(t, err)

	// Verify Output
	expectedOutput := "TorController version 1.1.0\n"
	assert.Equal(t, expectedOutput, output.String())
}
