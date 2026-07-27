package controller

// CheckTorIPv6Support now is NOT use, and is Wrong tor socket command.
// func (h *CommandHandler) CheckTorIPv6Support() (bool, error) {
// 	// Establish a connection to the Tor control port
// 	conn, err := h.Socket.Dial()
// 	if err != nil {
// 		h.Logger.Printf("[ERROR] Failed to connect to Tor control port: %v", err)
// 		return false, fmt.Errorf("failed to connect to Tor control port: %v", err)
// 	}
// 	defer conn.Close()

// 	// read control.authcookie
// 	cookie, err := os.ReadFile("/var/lib/tor/control.authcookie")
// 	if err != nil {
// 		h.Logger.Printf("[ERROR] Failed to read control.authcookie: %v", err)
// 		return false, fmt.Errorf("failed to read control.authcookie: %v", err)
// 	}

// 	authCommand := fmt.Sprintf("AUTHENTICATE %s\r\n", hex.EncodeToString(cookie))
// 	_, err = conn.Write([]byte(authCommand))
// 	if err != nil {
// 		h.Logger.Printf("[ERROR] Failed to send AUTHENTICATE command: %v", err)
// 		return false, fmt.Errorf("failed to send AUTHENTICATE command: %v", err)
// 	}
// 	h.Logger.Println("[INFO] AUTHENTICATE command sent.")

// 	reader := bufio.NewReader(conn)
// 	for {
// 		line, err := reader.ReadString('\n')
// 		if err != nil {
// 			h.Logger.Printf("[ERROR] Failed to read AUTHENTICATE response: %v", err)
// 			return false, fmt.Errorf("failed to read AUTHENTICATE response: %v", err)
// 		}
// 		line = strings.TrimSpace(line)
// 		h.Logger.Printf("[DEBUG] AUTHENTICATE response: %s", line)

// 		if line == "250 OK" {
// 			break // Successful authentication
// 		} else if strings.HasPrefix(line, "5") { // Failed authentication
// 			h.Logger.Printf("[ERROR] Authentication failed: %s", line)
// 			return false, fmt.Errorf("authentication failed: %s", line)
// 		}
// 	}

// 	// Ask for IPv6 support
// 	_, err = conn.Write([]byte("GETINFO ip-to-country/ipv6-available\r\n"))
// 	if err != nil {
// 		h.Logger.Printf("[ERROR] Failed to send GETINFO command: %v", err)
// 		return false, fmt.Errorf("failed to send GETINFO command: %v", err)
// 	}
// 	h.Logger.Println("[INFO] GETINFO ip-to-country/ipv6-available command sent.")

// 	for {
// 		line, err := reader.ReadString('\n')
// 		if err != nil {
// 			h.Logger.Printf("[ERROR] Failed to read GETINFO response: %v", err)
// 			return false, fmt.Errorf("failed to read GETINFO response: %v", err)
// 		}
// 		line = strings.TrimSpace(line)
// 		h.Logger.Printf("[DEBUG] GETINFO response: %s", line)

// 		if strings.HasPrefix(line, "250-ip-to-country/ipv6-available=") {
// 			value := strings.TrimPrefix(line, "250-ip-to-country/ipv6-available=")
// 			h.Logger.Printf("[INFO] IPv6 available: %s", value)
// 			return value == "1", nil // Returns true if the value is “1”, otherwise false.
// 		} else if strings.HasPrefix(line, "250 OK") {
// 			break // End response
// 		}
// 	}

// 	h.Logger.Println("[WARN] Unexpected response: failed to determine IPv6 support.")
// 	return false, fmt.Errorf("unexpected response: failed to determine IPv6 support")
// }
