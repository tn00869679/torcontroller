package initializer

import (
	"fmt"
)

// VerifyTorrcConfig validates the Torrc configuration.
func (i *Initializer) VerifyTorrcConfig() bool {
	cmd := []string{"sudo", "tor", "--verify-config"}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		fmt.Printf("- Torrc config validation failed: %v\n", err)
		return false
	}
	return true
}
