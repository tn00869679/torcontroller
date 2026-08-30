package controller

import (
	"fmt"
	"strconv"
	"strings"
)

// Socket states as reported by /proc/net. A TCP listener sits in 0A (LISTEN);
// an unconnected UDP socket, which is what a DNS listener is, sits in 07.
const (
	tcpStateListen = "0A"
	udpStateUnconn = "07"
)

// VerifyTorProxyPorts reports whether Tor is listening on the ports the
// redirection rules point at.
//
// This is the last guard before the rules go in. iptables happily redirects to
// a port with nothing behind it: connections are accepted by the kernel and
// then dropped, so the machine loses the network with no error anywhere. That
// is far worse than the feature simply not switching on, which is what
// returning an error here produces.
func (h *CommandHandler) VerifyTorProxyPorts(cfg ProxyConfig) error {
	checks := []struct {
		description string
		procFile    string
		port        string
		state       string
	}{
		{"TransPort (redirected TCP)", "/proc/net/tcp", TransPortIPv4, tcpStateListen},
		{"DNSPort (redirected DNS)", "/proc/net/udp", DNSPortUDP, udpStateUnconn},
	}

	// The IPv6 listener only matters when the IPv6 rules are going in. Demanding
	// it otherwise would block hosts that deliberately run IPv4 only.
	if cfg.EnableIPv6 {
		checks = append(checks, struct {
			description string
			procFile    string
			port        string
			state       string
		}{"TransPort over IPv6", "/proc/net/tcp6", TransPortIPv6, tcpStateListen})
	}

	var missing []string
	for _, check := range checks {
		listening, err := h.portIsListening(check.procFile, check.port, check.state)
		if err != nil {
			return fmt.Errorf("could not determine whether %s is ready: %w", check.description, err)
		}
		if !listening {
			missing = append(missing, fmt.Sprintf("%s on port %s", check.description, check.port))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"Tor is not listening on %s; check that torrc still contains the TransPort and DNSPort lines, then restart tor. Refusing to install redirection rules that would take the network offline",
			strings.Join(missing, " and "),
		)
	}

	h.Logger.Println("[INFO] Tor is listening on the transparent proxy ports.")
	return nil
}

// portIsListening scans a /proc/net table for a socket bound to port in the
// given state. /proc is used rather than ss or netstat because neither is a
// dependency of this package, and a missing tool must not be mistaken for a
// missing listener.
func (h *CommandHandler) portIsListening(procFile, port, state string) (bool, error) {
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return false, fmt.Errorf("invalid port %q: %w", port, err)
	}

	contents, err := h.FileSystem.ReadFile(procFile)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", procFile, err)
	}

	// local_address is "HEXADDR:HEXPORT"; the port is four uppercase hex
	// digits regardless of address family.
	wantSuffix := fmt.Sprintf(":%04X", portNumber)

	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if strings.HasSuffix(fields[1], wantSuffix) && fields[3] == state {
			return true, nil
		}
	}
	return false, nil
}
