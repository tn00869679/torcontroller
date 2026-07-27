package controller

import (
	"fmt"
	"strings"
)

// StopServiceFactory is a factory method for stopping and checking systemd services
func (h *CommandHandler) StopServiceFactory(service string) error {
	h.Logger.Printf("[INFO] Stopping %s service...", service)

	stopOutput, err := h.CommandRunner.Run("sudo", "systemctl", "stop", service)
	h.Logger.Printf("[INFO] %s service stop output: %s", service, stopOutput)

	if err != nil {
		if strings.Contains(stopOutput, "could not be found") {
			h.Logger.Printf("[WARN] %s service not found. Please install and configure %s.", service, service)
			return fmt.Errorf("%s service not found", service)
		}
		h.Logger.Printf("[ERROR] Failed to stop %s service: %v", service, err)
		return fmt.Errorf("failed to stop %s service: %w", service, err)
	}

	// Verify that the service is stopped
	h.Logger.Printf("[INFO] Verifying %s service status...", service)
	statusOutput, err := h.CommandRunner.Run("sudo", "systemctl", "status", service, "--no-pager")
	h.Logger.Printf("[INFO] %s service status output: %s", service, statusOutput)

	if err != nil {
		if strings.Contains(statusOutput, "inactive (dead)") {
			h.Logger.Printf("[INFO] %s service stopped successfully.", service)
			return nil
		}
		h.Logger.Printf("[ERROR] Failed to verify %s service status: %v", service, err)
		return fmt.Errorf("failed to verify %s service status: %w", service, err)
	}

	if strings.Contains(statusOutput, "inactive (dead)") {
		h.Logger.Printf("[INFO] %s service stopped successfully.", service)
		return nil
	}

	h.Logger.Printf("[WARN] %s service stop command issued, but service is still running.", service)
	return fmt.Errorf("%s service stop command issued, but service is still running", service)
}

// StopTorService stops the Tor service using the factory method
func (h *CommandHandler) StopTorService() error {
	return h.StopServiceFactory("tor")
}

// StopPrivoxyService stops the Privoxy service using the factory method
func (h *CommandHandler) StopPrivoxyService() error {
	return h.StopServiceFactory("privoxy")
}
