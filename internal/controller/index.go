package controller

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

func HandleConnection(conn net.Conn, socketPath string, listener net.Listener) error {
	fs := &RealFileSystem{}
	handler := NewCommandHandler(&RealSocket{Address: "127.0.0.1:9051"}, &RealCommandRunner{}, nil, nil, fs)
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			handler.Logger.Println("[WARN] Connection closed by client.")
			return nil
		}
		handler.Logger.Printf("[ERROR] Error reading from connection: %v", err)
		return err
	}

	command := strings.TrimSpace(string(buf[:n]))
	handler.Logger.Printf("[INFO] Received command on %s: %s", socketPath, command)

	switch command {
	case "start":
		if err := handler.StartTorService(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}

		if err := handler.StartPrivoxyService(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}

		if err := handler.ApplyIptablesIPv4(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}

		if err := handler.ApplyIptablesIPv6Reject(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}

		// Retrieve traffic metrics
		readTraffic, writtenTraffic, err := handler.GetTorTrafficMetrics()
		if err != nil {
			handler.Logger.Printf("[ERROR] Failed to retrieve traffic metrics: %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}

		// Check against configuration limits
		if (handler.Config.RateLimit.MinReadRate > 0 && readTraffic < handler.Config.RateLimit.MinReadRate) ||
			(handler.Config.RateLimit.MinWriteRate > 0 && writtenTraffic < handler.Config.RateLimit.MinWriteRate) {
			handler.Logger.Printf("[WARN] Traffic below threshold. Read: %d bytes (limit: %d bytes), Written: %d bytes (limit: %d bytes)",
				readTraffic, handler.Config.RateLimit.MinReadRate, writtenTraffic, handler.Config.RateLimit.MinWriteRate)

			if err := handler.SwitchTorCircuit(); err != nil {
				handler.Logger.Printf("[ERROR] Failed to switch Tor circuit: %v", err)
				_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
				return err
			}
		}

		_, _ = conn.Write([]byte("Done\n"))
		handler.Logger.Println("[INFO] Tor service started successfully.")
		return nil
	case "switch":
		if err := handler.SwitchTorCircuit(); err != nil {
			handler.Logger.Printf("[ERROR] Failed to switch Tor circuit: %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}
		_, _ = conn.Write([]byte("Circuit switched successfully\n"))
		handler.Logger.Println("[INFO] Successfully switched Tor circuit.")
		return nil
	case "traffic":
		readTraffic, writtenTraffic, err := handler.GetTorTrafficMetrics()
		if err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}
		response := fmt.Sprintf("Traffic Read: %d bytes, Traffic Written: %d bytes\n", readTraffic, writtenTraffic)
		if _, err := conn.Write([]byte(response)); err != nil {
			handler.Logger.Printf("[ERROR] Failed to send traffic response: %v", err)
			return err
		}
		handler.Logger.Printf("[INFO] Traffic Read: %d bytes, Traffic Written: %d bytes", readTraffic, writtenTraffic)
		return nil
	case "stop":
		if err := handler.ClearIptablesIPv6Reject(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}

		if err := handler.ClearIptablesIPv4(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}
		if err := handler.StopPrivoxyService(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}
		if err := handler.StopTorService(); err != nil {
			handler.Logger.Printf("[ERROR] %v", err)
			_, _ = conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			return err
		}
		_, _ = conn.Write([]byte("Done\n"))
		handler.Logger.Println("[INFO] Tor service stopped successfully.")
		return nil
	default:
		msg := fmt.Sprintf("Unknown command: %s\nAvailable commands: start, switch, traffic, stop\n", command)
		handler.Logger.Println("[WARN] " + msg)
		_, _ = conn.Write([]byte(msg))
		return nil
	}
}
