package cmd_test

import "net"

// Simulated ConnectionAdapter
type MockAdapter struct {
	Client net.Conn
	Server net.Conn
}

func (m *MockAdapter) Dial() (net.Conn, error) {
	return m.Client, nil
}
