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

func TestPortsVerifyPassesWhenTorIsListening(t *testing.T) {
	handler := handlerWithProc(procTCPListening, procUDPListening)

	if err := handler.VerifyTorProxyPorts(); err != nil {
		t.Fatalf("expected the check to pass, got: %v", err)
	}
}

// This is the guard that keeps a broken torrc from taking the machine off the
// network: rules aimed at a closed port drop every connection with no error.
func TestPortsVerifyRefusesWhenTransPortIsMissing(t *testing.T) {
	handler := handlerWithProc(procTCPEmpty, procUDPListening)

	err := handler.VerifyTorProxyPorts()
	if err == nil {
		t.Fatal("expected a refusal when TransPort is not listening")
	}
	if !strings.Contains(err.Error(), controller.TransPortIPv4) {
		t.Errorf("the error should name the missing port, got: %v", err)
	}
}

func TestPortsVerifyRefusesWhenDNSPortIsMissing(t *testing.T) {
	handler := handlerWithProc(procTCPListening, procUDPEmpty)

	err := handler.VerifyTorProxyPorts()
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

	if err := handler.VerifyTorProxyPorts(); err == nil {
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

	if err := handler.VerifyTorProxyPorts(); err == nil {
		t.Fatal("expected an error when /proc/net/tcp cannot be read")
	}
}
