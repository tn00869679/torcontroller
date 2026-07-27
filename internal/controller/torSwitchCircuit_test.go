package controller_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/Seicrypto/torcontroller/internal/controller"
	"github.com/Seicrypto/torcontroller/internal/singleton/configuration"
)

func TestSwitchTorCircuit(t *testing.T) {
	// Mock Logger
	mockLogger := NewMockLogger()
	closeSignal := make(chan struct{})

	// Generate the correct hex-encoded string for the mock cookie content
	mockEncodedCookie := "880952c76d34c35ea30eac0f2bb40bfa4ffcf4a05e8e76724bfd6c6bc1efb23a"
	decodedCookie, err := hex.DecodeString(mockEncodedCookie) // Correctly decode the mock cookie
	if err != nil {
		t.Fatalf("unexpected mockEncodedCookie: %v", err)
	}

	// Mock Socket
	mockSocket := &MockSocket{
		ResponseMap: map[string]string{
			fmt.Sprintf("AUTHENTICATE %s\r\n", mockEncodedCookie): "250 OK\n",
			"SIGNAL NEWNYM\r\n":           "250 OK\n",
			"GETINFO traffic/read\r\n":    "250-traffic/read=500\n250 OK\n",
			"GETINFO traffic/written\r\n": "250-traffic/written=300\n250 OK\n",
		},
		CloseSignal: closeSignal,
	}

	// Mock FileSystem
	mockFileSystem := &MockFileSystem{
		Files: map[string]*MockFileInfo{
			"/var/lib/tor/control.authcookie": {
				content: decodedCookie, // Use the raw mock cookie content
				mode:    0644,          // Typical file mode
				uid:     0,             // Root ownership
				gid:     0,             // Root group
			},
		},
		Error: nil, // No error to simulate
	}

	// Mock CommandRunner (not used in this test but provided for completeness)
	mockRunner := &MockCommandRunner{}

	// Mock Config with thresholds
	mockConfig := &configuration.Configuration{
		RateLimit: configuration.RateLimitConfig{
			MinReadRate:  600,
			MinWriteRate: 400,
		},
	}

	// Create CommandHandler
	handler := controller.NewCommandHandler(mockSocket, mockRunner, mockLogger, mockConfig, mockFileSystem)

	// Execute the method
	err = handler.SwitchTorCircuit()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Signal to close the MockSocket
	close(closeSignal)

	// Verify logger output for expected behavior
	buffer := mockLogger.Writer().(*bytes.Buffer)
	logOutput := buffer.String()

	// Expected log checks
	if !strings.Contains(logOutput, "AUTHENTICATE command sent") {
		t.Error("Expected 'AUTHENTICATE command sent' log not found")
	}
	if !strings.Contains(logOutput, "SIGNAL NEWNYM command sent") {
		t.Error("Expected 'SIGNAL NEWNYM command sent' log not found")
	}
	if strings.Contains(logOutput, "Traffic metrics are within acceptable limits") {
		t.Error("Unexpected 'Traffic metrics are within acceptable limits' log found")
	}
	if !strings.Contains(logOutput, "Traffic below threshold, sending SIGNAL NEWNYM") {
		t.Error("Expected 'Traffic below threshold, sending SIGNAL NEWNYM' log not found")
	}
	if !strings.Contains(logOutput, "Tor circuit switched successfully") {
		t.Error("Expected 'Tor circuit switched successfully' log not found")
	}
}
