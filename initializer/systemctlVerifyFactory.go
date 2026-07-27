package initializer

import (
	"fmt"
	"strings"
)

// CheckTorService checks the validity of the Tor service unit file.
func (i *Initializer) CheckTorService() bool {
	return i.CheckServiceFile("tor")
}

// CheckPrivoxyService checks the validity of the Privoxy service unit file.
func (i *Initializer) CheckPrivoxyService() bool {
	return i.CheckServiceFile("privoxy")
}

// CheckServiceFile validates the given service's systemd unit file.
func (i *Initializer) CheckServiceFile(serviceName string) bool {
	cmd := []string{"sudo", "systemctl", "show", serviceName}
	output, err := i.CommandRunner.Run(cmd[0], cmd[1:]...)
	if err != nil {
		fmt.Printf("[ERROR] Failed to validate service %s: %v\n", serviceName, err)
		return false
	}

	// Parse the output to check for critical fields
	if !strings.Contains(output, "LoadState=loaded") {
		fmt.Printf("[ERROR] Service %s is not loaded properly.\n", serviceName)
		return false
	}

	// Additional checks can be added here if needed
	return true
}
