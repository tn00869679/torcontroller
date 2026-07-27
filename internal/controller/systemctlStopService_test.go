package controller_test

import (
	"fmt"
	"testing"

	"github.com/Seicrypto/torcontroller/internal/controller"
)

func TestStopServiceFactory(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "stop", "tor"},
		"", nil) // Simulate a successful service shutdown

	mockRunner.On("sudo", []string{"systemctl", "status", "tor", "--no-pager"},
		"inactive (dead)\n", fmt.Errorf("inactive")) // The simulation rechecks the service status as stopped.

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StopTorService()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestStopServiceFactory_ServiceAlreadyStopped(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "stop", "tor"},
		"", nil) // Simulate stopping the service, even if it has been stopped without error.

	mockRunner.On("sudo", []string{"systemctl", "status", "tor", "--no-pager"},
		"inactive (dead)\n", fmt.Errorf("inactive")) // The simulation service has been stopped.

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StopTorService()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestStopServiceFactory_ServiceNotFound(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "stop", "nonexistent-service"},
		"could not be found\n", fmt.Errorf("not found")) // Simulation not found

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StopServiceFactory("nonexistent-service")
	if err == nil || err.Error() != "nonexistent-service service not found" {
		t.Fatalf("expected 'service not found' error, got: %v", err)
	}
}

func TestStopServiceFactory_FailToStop(t *testing.T) {
	mockLogger := NewMockLogger()
	mockRunner := &MockCommandRunner{}

	mockRunner.On("sudo", []string{"systemctl", "stop", "tor"},
		"", fmt.Errorf("failed to stop")) // Simulation stop service failure

	handler := controller.NewCommandHandler(nil, mockRunner, mockLogger, nil, nil)

	err := handler.StopTorService()
	if err == nil || err.Error() != "failed to stop tor service: failed to stop" {
		t.Fatalf("expected 'failed to stop' error, got: %v", err)
	}
}
