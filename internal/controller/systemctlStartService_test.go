package controller_test

import (
	"fmt"
	"testing"

	"github.com/Seicrypto/torcontroller/internal/controller"
)

func TestStartServiceFactory(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "status", "tor", "--no-pager"},
		"inactive (dead)\n", fmt.Errorf("inactive")) // The simulation service is stopped.

	mockRunner.On("sudo", []string{"systemctl", "start", "tor"}, "", nil) // Successful simulation starts the service

	mockRunner.On("sudo", []string{"systemctl", "status", "tor", "--no-pager"},
		"active (running)\n", nil) // The simulation rechecks the service status as running.

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StartTorService()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestStartServiceFactory_ServiceAlreadyRunning(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "status", "tor", "--no-pager"},
		"active (running)\n", fmt.Errorf("already running")) // The simulation service is already running

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StartTorService()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestStartServiceFactory_ServiceNotFound(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "status", "nonexistent-service", "--no-pager"},
		"could not be found\n", fmt.Errorf("not found")) // Simulation not found

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StartServiceFactory("nonexistent-service")
	if err == nil || err.Error() != "nonexistent-service service not found" {
		t.Fatalf("expected 'service not found' error, got: %v", err)
	}
}

func TestStartServiceFactory_FailToStart(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "status", "tor", "--no-pager"},
		"inactive (dead)\n", fmt.Errorf("inactive")) // The simulation service is stopped.

	mockRunner.On("sudo", []string{"systemctl", "start", "tor"}, "", fmt.Errorf("failed to start")) // Simulation startup failed

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StartTorService()
	if err == nil || err.Error() != "failed to start tor service: failed to start" {
		t.Fatalf("expected 'failed to start' error, got: %v", err)
	}
}
