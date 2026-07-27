package controller_test

import (
	"testing"
	"time"
)

func TestMockSocket(t *testing.T) {
	mockResponseMap := map[string]string{
		"AUTHENTICATE 1234567890abcdef1234567890abcdef\r\n": "250 OK\n",
		"GETINFO traffic/read\r\n":                          "250-traffic/read=5000\n250 OK\n",
		"GETINFO traffic/written\r\n":                       "250-traffic/written=3000\n250 OK\n",
	}

	mockSocket := &MockSocket{
		ResponseMap: mockResponseMap,
		Error:       nil,
	}

	conn, err := mockSocket.Dial()
	if err != nil {
		t.Fatalf("failed to dial mock socket: %v", err)
	}
	defer conn.Close()

	tests := []struct {
		command          string
		expectedResponse string
	}{
		{"AUTHENTICATE 1234567890abcdef1234567890abcdef\r\n", "250 OK\n"},
		{"GETINFO traffic/read\r\n", "250-traffic/read=5000\n250 OK\n"},
		{"GETINFO traffic/written\r\n", "250-traffic/written=3000\n250 OK\n"},
		{"UNKNOWN COMMAND\r\n", "5XX Command not recognized\n"},
	}

	for _, tt := range tests {
		_, err := conn.Write([]byte(tt.command))
		if err != nil {
			t.Fatalf("failed to write command: %v", err)
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		response := string(buf[:n])
		if response != tt.expectedResponse {
			t.Errorf("unexpected response for command %q: got %q, want %q", tt.command, response, tt.expectedResponse)
		}
	}
}

func TestMockSocketWithCloseSignal(t *testing.T) {
	// Create a close signal channel
	closeSignal := make(chan struct{})

	mockSocket := &MockSocket{
		ResponseMap: map[string]string{
			"AUTHENTICATE 1234567890abcdef1234567890abcdef\r\n": "250 OK\n",
		},
		CloseSignal: closeSignal,
	}

	// Dial the mock socket
	conn, err := mockSocket.Dial()
	if err != nil {
		t.Fatalf("failed to dial mock socket: %v", err)
	}
	defer conn.Close()

	// Simulate sending a command
	_, err = conn.Write([]byte("AUTHENTICATE 1234567890abcdef1234567890abcdef\r\n"))
	if err != nil {
		t.Fatalf("failed to write to mock socket: %v", err)
	}

	// Read the response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from mock socket: %v", err)
	}
	response := string(buf[:n])
	expected := "250 OK\n"
	if response != expected {
		t.Errorf("unexpected response: got %q, want %q", response, expected)
	}

	// Trigger the close signal
	close(closeSignal)

	// Wait briefly to ensure the goroutine has time to exit
	time.Sleep(100 * time.Millisecond)

	// Verify that the goroutine has exited (by ensuring no new responses are possible)
	_, err = conn.Write([]byte("AUTHENTICATE anothercommand\r\n"))
	if err == nil {
		t.Errorf("expected write to fail after socket is closed")
	}
}
