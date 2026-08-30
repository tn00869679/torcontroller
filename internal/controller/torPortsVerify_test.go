package controller_test

import (
	"strings"
	"testing"

	"github.com/tn00869679/torcontroller/internal/controller"
)

// /proc/net rows as the kernel writes them. Ports are four hex digits:
// 9040 is 2350 and 5353 is 14E9. The state column is 0A for a TCP listener
// and 07 for an unconnected UDP socket.
const (
	procTCPListening = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:2350 00000000:0000 0A 00000000:00000000 00:00000000 00000000   102        0 24601 1
   1: 0100007F:235B 00000000:0000 0A 00000000:00000000 00:00000000 00000000   102        0 24602 1
`
	procTCPEmpty = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
`
	procUDPListening = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
  456: 0100007F:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000   102        0 24603 2
`
	procUDPEmpty = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
`
	// Same port number, but the socket is connected rather than listening.
	procTCPConnected = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:2350 08080808:0050 01 00000000:00000000 00:00000000 00000000   102        0 24601 1
`
	// /proc/net/tcp6 uses a 32-character address; 9041 is 2351. This row is
	// Tor's TransPort bound to [::1].
	procTCP6Listening = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000001000000:2351 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000   102        0 24604 1
`
	procTCP6Empty = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
`
)

func handlerWithProc(tcp, udp string) *controller.CommandHandler {
	return &controller.CommandHandler{
		Logger: NewMockLogger(),
		FileSystem: &MockFileSystem{
			Files: map[string]*MockFileInfo{
				"/proc/net/tcp": {content: []byte(tcp)},
				"/proc/net/udp": {content: []byte(udp)},
			},
		},
	}
}

// Most of these cases are about the IPv4 listeners, so they run with IPv6
// switched off to keep the IPv6 listener out of the assertion.
func ipv4OnlyConfig() controller.ProxyConfig {
	cfg := controller.DefaultProxyConfig("102")
	cfg.EnableIPv6 = false
	return cfg
}

func handlerWithProc6(tcp, udp, tcp6 string) *controller.CommandHandler {
	return &controller.CommandHandler{
		Logger: NewMockLogger(),
		FileSystem: &MockFileSystem{
			Files: map[string]*MockFileInfo{
				"/proc/net/tcp":  {content: []byte(tcp)},
				"/proc/net/udp":  {content: []byte(udp)},
				"/proc/net/tcp6": {content: []byte(tcp6)},
			},
		},
	}
}

// With IPv6 enabled the rules redirect to TransPort 9041 as well, so that
// listener has to exist before they go in.
func TestPortsVerifyRequiresTheIPv6ListenerOnlyWhenIPv6IsEnabled(t *testing.T) {
	handler := handlerWithProc6(procTCPListening, procUDPListening, procTCP6Empty)

	if err := handler.VerifyTorProxyPorts(ipv4OnlyConfig()); err != nil {
		t.Fatalf("an IPv4-only run must not demand the IPv6 listener: %v", err)
	}

	cfg := controller.DefaultProxyConfig("102")
	err := handler.VerifyTorProxyPorts(cfg)
	if err == nil {
		t.Fatal("expected a refusal when the IPv6 TransPort is missing")
	}
	if !strings.Contains(err.Error(), controller.TransPortIPv6) {
		t.Errorf("the error should name port %s, got: %v", controller.TransPortIPv6, err)
	}
}

func TestPortsVerifyPassesWithBothFamiliesListening(t *testing.T) {
	handler := handlerWithProc6(procTCPListening, procUDPListening, procTCP6Listening)

	if err := handler.VerifyTorProxyPorts(controller.DefaultProxyConfig("102")); err != nil {
		t.Fatalf("expected the check to pass, got: %v", err)
	}
}

func TestPortsVerifyPassesWhenTorIsListening(t *testing.T) {
	handler := handlerWithProc(procTCPListening, procUDPListening)

	if err := handler.VerifyTorProxyPorts(ipv4OnlyConfig()); err != nil {
		t.Fatalf("expected the check to pass, got: %v", err)
	}
}

// This is the guard that keeps a broken torrc from taking the machine off the
// network: rules aimed at a closed port drop every connection with no error.
func TestPortsVerifyRefusesWhenTransPortIsMissing(t *testing.T) {
	handler := handlerWithProc(procTCPEmpty, procUDPListening)

	err := handler.VerifyTorProxyPorts(ipv4OnlyConfig())
	if err == nil {
		t.Fatal("expected a refusal when TransPort is not listening")
	}
	if !strings.Contains(err.Error(), controller.TransPortIPv4) {
		t.Errorf("the error should name the missing port, got: %v", err)
	}
}

func TestPortsVerifyRefusesWhenDNSPortIsMissing(t *testing.T) {
	handler := handlerWithProc(procTCPListening, procUDPEmpty)

	err := handler.VerifyTorProxyPorts(ipv4OnlyConfig())
	if err == nil {
		t.Fatal("expected a refusal when DNSPort is not listening")
	}
	if !strings.Contains(err.Error(), controller.DNSPortUDP) {
		t.Errorf("the error should name the missing port, got: %v", err)
	}
}

// A socket on the right port that is connected rather than listening cannot
// accept the redirected traffic, so matching on the port alone is not enough.
func TestPortsVerifyIgnoresSocketsThatAreNotListening(t *testing.T) {
	handler := handlerWithProc(procTCPConnected, procUDPListening)

	if err := handler.VerifyTorProxyPorts(ipv4OnlyConfig()); err == nil {
		t.Fatal("a connected socket on the same port must not count as a listener")
	}
}

// An unreadable /proc must not be mistaken for a healthy system; failing to
// determine the state has to stop the rules going in.
func TestPortsVerifyFailsWhenProcCannotBeRead(t *testing.T) {
	handler := &controller.CommandHandler{
		Logger:     NewMockLogger(),
		FileSystem: &MockFileSystem{Files: map[string]*MockFileInfo{}},
	}

	if err := handler.VerifyTorProxyPorts(ipv4OnlyConfig()); err == nil {
		t.Fatal("expected an error when /proc/net/tcp cannot be read")
	}
}
