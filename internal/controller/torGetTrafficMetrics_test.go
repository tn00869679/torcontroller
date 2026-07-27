package controller_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/Seicrypto/torcontroller/internal/controller"
	"github.com/Seicrypto/torcontroller/internal/singleton/configuration"
)

func TestGetTorTrafficMetrics(t *testing.T) {
	mockLogger := NewMockLogger()
	closeSignal := make(chan struct{})

	// Generate the correct hex-encoded string for the mock cookie content
	mockEncodedCookie := "880952c76d34c35ea30eac0f2bb40bfa4ffcf4a05e8e76724bfd6c6bc1efb23a"
	decodedCookie, err := hex.DecodeString(mockEncodedCookie) // Correctly decode the mock cookie
	if err != nil {
		t.Fatalf("unexpected mockEncodedCookie: %v", err)
	}
	if len(decodedCookie) != 32 {
		t.Fatalf("unexpected decodedCookie length: got %d, want 32", len(decodedCookie))
	}

	mockSocket := &MockSocket{
		ResponseMap: map[string]string{
			fmt.Sprintf("AUTHENTICATE %s\r\n", mockEncodedCookie): "250 OK\n",
			"GETINFO traffic/read\r\n":                            "250-traffic/read=5000\n250 OK\n",
			"GETINFO traffic/written\r\n":                         "250-traffic/written=3000\n250 OK\n",
		},
		CloseSignal: closeSignal,
	}
	mockRunner := &MockCommandRunner{}
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

	mockConfig := configuration.MockConfiguration(500, 300)
	handler := controller.NewCommandHandler(mockSocket, mockRunner, mockLogger, mockConfig, mockFileSystem)

	readTraffic, writtenTraffic, err := handler.GetTorTrafficMetrics()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if readTraffic != 5000 {
		t.Errorf("expected read traffic to be 5000, got %d", readTraffic)
	}

	if writtenTraffic != 3000 {
		t.Errorf("expected written traffic to be 3000, got %d", writtenTraffic)
	}

	// Signal to close the MockSocket
	close(closeSignal)
}
