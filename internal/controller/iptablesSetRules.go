package controller

import (
	"fmt"
	"strings"
)

// Ports Tor listens on for transparently redirected traffic. They must match
// the TransPort and DNSPort lines in the shipped torrc: a rule pointing at a
// port nothing listens on does not fail loudly, it silently blackholes every
// connection the machine makes.
const (
	TransPortIPv4 = "9040"
	DNSPortUDP    = "5353"

	// TorChain holds every rule we install. Keeping them out of OUTPUT makes
	// teardown "unhook and flush" instead of deleting rules one at a time,
	// which used to leave half-applied state behind after the first failure
	// and could delete an identical rule the operator had added themselves.
	TorChain = "TORCONTROLLER"

	// TorUser is the account Debian's tor package runs the daemon as.
	TorUser = "debian-tor"
)

// Rule is a single iptables invocation.
type Rule struct {
	Command string
	Args    []string
}

// ProxyConfig carries the values that differ between hosts.
type ProxyConfig struct {
	// TorUID is the uid Tor runs as. Its own connections to relays must skip
	// redirection, or Tor is redirected into itself and nothing reaches the
	// network at all. The uid has to be dedicated: exempting root would exempt
	// every privileged process on the machine and reopen the leak this is
	// meant to close.
	TorUID string

	// VirtualNetIPv4 is the range AutomapHostsOnResolve draws from. It must
	// match VirtualAddrNetworkIPv4 in torrc.
	VirtualNetIPv4 string

	// ExcludedNets are destinations that stay off Tor: loopback so local
	// services keep working, and the RFC1918 ranges so the LAN stays
	// reachable.
	ExcludedNets []string
}

// DefaultProxyConfig returns the settings used when nothing overrides them.
func DefaultProxyConfig(torUID string) ProxyConfig {
	return ProxyConfig{
		TorUID:         torUID,
		VirtualNetIPv4: "10.192.0.0/10",
		ExcludedNets: []string{
			"127.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		},
	}
}

// BuildIPv4Rules returns the chain contents in the order iptables evaluates
// them. The order is the design:
//
//  1. Tor's own traffic returns first, before anything can redirect it.
//  2. The automap range is redirected before the private-network exemptions,
//     because the default virtual range sits inside 10.0.0.0/8 and would
//     otherwise be skipped -- every hostname resolved through Tor would then
//     bypass Tor when connected to.
//  3. DNS is redirected before the loopback exemption, so queries aimed at a
//     resolver on 127.0.0.53 are still caught.
//  4. Loopback and LAN return.
//  5. Everything else is redirected. Matching all TCP rather than ports 80
//     and 443 is what puts SSH, mail and every other protocol through Tor.
func BuildIPv4Rules(cfg ProxyConfig) []Rule {
	appendToChain := func(args ...string) Rule {
		return Rule{
			Command: "iptables",
			Args:    append([]string{"-t", "nat", "-A", TorChain}, args...),
		}
	}

	rules := []Rule{
		appendToChain("-m", "owner", "--uid-owner", cfg.TorUID, "-j", "RETURN"),
		appendToChain("-p", "tcp", "-d", cfg.VirtualNetIPv4, "-j", "REDIRECT", "--to-ports", TransPortIPv4),
		appendToChain("-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", DNSPortUDP),
	}
	for _, network := range cfg.ExcludedNets {
		rules = append(rules, appendToChain("-d", network, "-j", "RETURN"))
	}
	return append(rules, appendToChain("-p", "tcp", "-j", "REDIRECT", "--to-ports", TransPortIPv4))
}

// TorUID looks up the uid Tor runs as. It is read rather than hardcoded
// because the number is assigned at package install time and differs between
// systems.
func (h *CommandHandler) TorUID() (string, error) {
	output, err := h.CommandRunner.Run("id", "-u", TorUser)
	if err != nil {
		return "", fmt.Errorf("failed to look up uid of %s (is the tor package installed?): %w", TorUser, err)
	}
	uid := strings.TrimSpace(output)
	if uid == "" {
		return "", fmt.Errorf("uid lookup for %s returned nothing", TorUser)
	}
	return uid, nil
}

// runRule executes one iptables invocation.
func (h *CommandHandler) runRule(rule Rule) error {
	command := strings.Join(append([]string{rule.Command}, rule.Args...), " ")
	h.Logger.Printf("[INFO] Applying rule: %s", command)

	output, err := h.CommandRunner.Run("sudo", append([]string{rule.Command}, rule.Args...)...)
	if err != nil {
		h.Logger.Printf("[ERROR] Rule failed: %s. Error: %s", command, err.Error())
		return fmt.Errorf("failed to apply iptables rule %q: %w", command, err)
	}
	if output != "" {
		h.Logger.Printf("[INFO] Command output: %s", output)
	}
	return nil
}

// ApplyTransparentProxy installs the redirection chain and hooks it into
// OUTPUT.
//
// The chain is built in full before the jump is added. Until that last step
// the chain is unreachable, so a failure part-way through cannot leave traffic
// partially redirected -- the chain is simply torn down and the host is
// exactly as it was.
func (h *CommandHandler) ApplyTransparentProxy(cfg ProxyConfig) error {
	if cfg.TorUID == "" {
		return fmt.Errorf("refusing to install rules without Tor's uid: every connection including Tor's own would be redirected into Tor")
	}

	create := Rule{Command: "iptables", Args: []string{"-t", "nat", "-N", TorChain}}
	if err := h.runRule(create); err != nil {
		// A leftover chain from an interrupted run is the likely cause. Flush
		// it and carry on rather than refusing to start.
		h.Logger.Printf("[WARN] Could not create chain %s, flushing an existing one instead.", TorChain)
		flush := Rule{Command: "iptables", Args: []string{"-t", "nat", "-F", TorChain}}
		if err := h.runRule(flush); err != nil {
			return fmt.Errorf("chain %s is neither creatable nor flushable: %w", TorChain, err)
		}
	}

	for _, rule := range BuildIPv4Rules(cfg) {
		if err := h.runRule(rule); err != nil {
			h.Logger.Printf("[ERROR] Rule set incomplete, removing chain %s before it takes effect.", TorChain)
			h.ClearTransparentProxy()
			return err
		}
	}

	hook := Rule{Command: "iptables", Args: []string{"-t", "nat", "-A", "OUTPUT", "-j", TorChain}}
	if err := h.runRule(hook); err != nil {
		h.ClearTransparentProxy()
		return err
	}

	h.Logger.Println("[INFO] Transparent proxy rules applied successfully.")
	return nil
}

// ClearTransparentProxy unhooks and removes the chain.
//
// Every step is best-effort: stop has to work when apply failed half-way, when
// the chain was never created, and when it was already removed. Refusing to
// continue after the first error would strand the caller with rules in place
// and no way to remove them.
func (h *CommandHandler) ClearTransparentProxy() error {
	steps := []Rule{
		{Command: "iptables", Args: []string{"-t", "nat", "-D", "OUTPUT", "-j", TorChain}},
		{Command: "iptables", Args: []string{"-t", "nat", "-F", TorChain}},
		{Command: "iptables", Args: []string{"-t", "nat", "-X", TorChain}},
	}

	var failures []string
	for _, rule := range steps {
		if err := h.runRule(rule); err != nil {
			failures = append(failures, strings.Join(rule.Args, " "))
		}
	}

	if len(failures) == len(steps) {
		// Nothing succeeded, so there was most likely nothing to remove.
		h.Logger.Printf("[INFO] No %s chain to remove.", TorChain)
		return nil
	}

	h.Logger.Println("[INFO] Transparent proxy rules cleared.")
	return nil
}
