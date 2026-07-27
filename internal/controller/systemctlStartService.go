package controller

import (
	"fmt"
	"strings"
)

// StartServiceFactory is a factory method for starting and checking systemd services
func (h *CommandHandler) StartServiceFactory(service string) error {
	h.Logger.Printf("[INFO] Checking %s service status...", service)

	statusOutput, err := h.CommandRunner.Run("sudo", "systemctl", "status", service, "--no-pager")
	// Log stdout regardless of error
	h.Logger.Printf("[INFO] %s service status output: %s", service, statusOutput)
	if err != nil {
		h.Logger.Printf("[INFO] %s service status output: %s", service, err)

		if strings.Contains(statusOutput, "inactive (dead)") {
			h.Logger.Printf("[WARN] %s service is inactive. Attempting to start...", service)
		} else if strings.Contains(statusOutput, "could not be found") {
			h.Logger.Printf("[ERROR] %s service not found. Please install and configure %s.", service, service)
			return fmt.Errorf("%s service not found", service)
		} else if strings.Contains(statusOutput, "active (running)") {
			h.Logger.Printf("%s service is already running.", service)
			return nil
		} else {
			h.Logger.Printf("[ERROR] Failed to check %s service status: %v", service, err)
			return fmt.Errorf("failed to check %s service status: %w", service, err)
		}
	}

	h.Logger.Printf("[INFO] Starting %s service...", service)
	_, err = h.CommandRunner.Run("sudo", "systemctl", "start", service)
	if err != nil {
		h.Logger.Printf("[ERROR] Failed to start %s service: %v", service, err)
		return fmt.Errorf("failed to start %s service: %w", service, err)
	}

	h.Logger.Printf("[INFO] Rechecking %s service status...", service)
	recheckOutput, err := h.CommandRunner.Run("sudo", "systemctl", "status", service, "--no-pager")

	if err != nil {
		if strings.Contains(recheckOutput, "active (running)") {
			h.Logger.Printf("[INFO] %s service is now running.", service)
			return nil
		} else if strings.Contains(err.Error(), "already started") {
			h.Logger.Printf("[INFO] %s service already started.", service)
			return nil
		}
		h.Logger.Printf("[ERROR] Failed to recheck %s service status: %v", service, err)
		return fmt.Errorf("failed to recheck %s service status: %w", service, err)
	}

	if strings.Contains(recheckOutput, "active (running)") {
		h.Logger.Printf("[INFO] %s service started successfully.", service)
		return nil
	}

	h.Logger.Printf("[ERROR] %s service failed to start, still not running.", service)
	return fmt.Errorf("%s service failed to start, still not running", service)
}

// StartTorService starts the Tor service using the factory method
func (h *CommandHandler) StartTorService() error {
	return h.StartServiceFactory("tor")
}

// StartPrivoxyService starts the Privoxy service using the factory method
func (h *CommandHandler) StartPrivoxyService() error {
	return h.StartServiceFactory("privoxy")
}
