package cmd

import (
	"fmt"
	"net"
)

// Defining a Connection Interface
type ConnectionAdapter interface {
	Dial() (net.Conn, error)
}

// UnixSocketAdapter for real Unix sockets
type UnixSocketAdapter struct {
	SocketPath string
}

func (u *UnixSocketAdapter) Dial() (net.Conn, error) {
	return net.Dial("unix", u.SocketPath)
}

// SocketInteractionHandler handling interactions with sockets
type SocketInteractionHandler struct {
	Adapter ConnectionAdapter
}

func (socket *SocketInteractionHandler) SendCommandAndGetResponse(command string) (string, error) {
	conn, err := socket.Adapter.Dial()
	if err != nil {
		return "", fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %v", err)
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	return string(buf[:n]), nil
}
