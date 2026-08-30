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
	TransPortIPv6 = "9041"
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

	// VirtualNetIPv4 and VirtualNetIPv6 are the ranges AutomapHostsOnResolve
	// draws from. They must match VirtualAddrNetworkIPv4 and
	// VirtualAddrNetworkIPv6 in torrc.
	//
	// The IPv6 default overlaps the range real unique-local networks use. A
	// host with an IPv6 LAN should narrow it, or those LAN addresses become
	// unreachable once the local route below is installed.
	VirtualNetIPv4 string
	VirtualNetIPv6 string

	// ExcludedNets and ExcludedNetsIPv6 stay off Tor: loopback so local
	// services keep working, and the LAN ranges so nearby machines stay
	// reachable.
	ExcludedNets     []string
	ExcludedNetsIPv6 []string

	// EnableIPv6 turns the whole IPv6 half off. Tor answers AAAA queries with
	// virtual addresses whenever automap is on, so leaving IPv6 unhandled is
	// not neutral: applications may prefer an address with nowhere to go.
	// Turning this off is therefore a deliberate fallback, not a default.
	EnableIPv6 bool
}

// DefaultProxyConfig returns the settings used when nothing overrides them.
func DefaultProxyConfig(torUID string) ProxyConfig {
	return ProxyConfig{
		TorUID:         torUID,
		VirtualNetIPv4: "10.192.0.0/10",
		VirtualNetIPv6: "fc00::/7",
		ExcludedNets: []string{
			"127.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		},
		ExcludedNetsIPv6: []string{
			"::1/128",
			"fe80::/10",
		},
		EnableIPv6: true,
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

// BuildIPv6Rules mirrors the IPv4 set. DNS is absent because queries reach the
// resolver over IPv4 here and are already caught by the IPv4 chain.
func BuildIPv6Rules(cfg ProxyConfig) []Rule {
	appendToChain := func(args ...string) Rule {
		return Rule{
			Command: "ip6tables",
			Args:    append([]string{"-t", "nat", "-A", TorChain}, args...),
		}
	}

	rules := []Rule{
		appendToChain("-m", "owner", "--uid-owner", cfg.TorUID, "-j", "RETURN"),
		appendToChain("-p", "tcp", "-d", cfg.VirtualNetIPv6, "-j", "REDIRECT", "--to-ports", TransPortIPv6),
	}
	for _, network := range cfg.ExcludedNetsIPv6 {
		rules = append(rules, appendToChain("-d", network, "-j", "RETURN"))
	}
	return append(rules, appendToChain("-p", "tcp", "-j", "REDIRECT", "--to-ports", TransPortIPv6))
}

// virtualRouteArgs builds the `ip -6 route` invocation for the automap range.
//
// Without this route the whole IPv6 half is inert. A connection to a virtual
// address fails in the kernel's routing lookup, which runs before the nat
// OUTPUT chain, so no redirect rule ever sees the packet. Marking the range
// local is what makes the packet exist for the rules to act on.
func virtualRouteArgs(action, network string) []string {
	return []string{"-6", "route", action, "local", network, "dev", "lo"}
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

// runRule executes one command, reporting failure.
func (h *CommandHandler) runRule(rule Rule) error {
	command := strings.Join(append([]string{rule.Command}, rule.Args...), " ")
	h.Logger.Printf("[INFO] Applying rule: %s", command)

	output, err := h.CommandRunner.Run("sudo", append([]string{rule.Command}, rule.Args...)...)
	if err != nil {
		h.Logger.Printf("[ERROR] Rule failed: %s. Error: %s", command, err.Error())
		return fmt.Errorf("failed to apply rule %q: %w", command, err)
	}
	if output != "" {
		h.Logger.Printf("[INFO] Command output: %s", output)
	}
	return nil
}

// tryRule runs a command whose failure is expected and harmless.
func (h *CommandHandler) tryRule(rule Rule) {
	if err := h.runRule(rule); err != nil {
		h.Logger.Printf("[INFO] Ignoring expected failure: %v", err)
	}
}

// prepareChain creates the chain, or empties an existing one left behind by an
// interrupted run.
func (h *CommandHandler) prepareChain(command string) error {
	create := Rule{Command: command, Args: []string{"-t", "nat", "-N", TorChain}}
	if err := h.runRule(create); err == nil {
		return nil
	}

	h.Logger.Printf("[WARN] Could not create %s chain %s, flushing an existing one instead.", command, TorChain)
	flush := Rule{Command: command, Args: []string{"-t", "nat", "-F", TorChain}}
	if err := h.runRule(flush); err != nil {
		return fmt.Errorf("%s chain %s is neither creatable nor flushable: %w", command, TorChain, err)
	}
	return nil
}

// ApplyTransparentProxy installs the redirection chains and hooks them into
// OUTPUT.
//
// Both chains are built in full before either jump is added. Until then the
// chains are unreachable, so a failure part-way through cannot leave traffic
// partially redirected -- the chains are torn down and the host is exactly as
// it was.
func (h *CommandHandler) ApplyTransparentProxy(cfg ProxyConfig) error {
	if cfg.TorUID == "" {
		return fmt.Errorf("refusing to install rules without Tor's uid: every connection including Tor's own would be redirected into Tor")
	}

	if err := h.prepareChain("iptables"); err != nil {
		return err
	}
	for _, rule := range BuildIPv4Rules(cfg) {
		if err := h.runRule(rule); err != nil {
			h.Logger.Printf("[ERROR] Rule set incomplete, removing chains before they take effect.")
			h.ClearTransparentProxy(cfg)
			return err
		}
	}

	if cfg.EnableIPv6 {
		// Replace rather than add: a route left behind by an interrupted run
		// would otherwise make this fail, or stack a duplicate.
		h.tryRule(Rule{Command: "ip", Args: virtualRouteArgs("del", cfg.VirtualNetIPv6)})
		if err := h.runRule(Rule{Command: "ip", Args: virtualRouteArgs("add", cfg.VirtualNetIPv6)}); err != nil {
			h.ClearTransparentProxy(cfg)
			return err
		}

		if err := h.prepareChain("ip6tables"); err != nil {
			h.ClearTransparentProxy(cfg)
			return err
		}
		for _, rule := range BuildIPv6Rules(cfg) {
			if err := h.runRule(rule); err != nil {
				h.Logger.Printf("[ERROR] IPv6 rule set incomplete, removing chains before they take effect.")
				h.ClearTransparentProxy(cfg)
				return err
			}
		}
	}

	hooks := []Rule{{Command: "iptables", Args: []string{"-t", "nat", "-A", "OUTPUT", "-j", TorChain}}}
	if cfg.EnableIPv6 {
		hooks = append(hooks, Rule{Command: "ip6tables", Args: []string{"-t", "nat", "-A", "OUTPUT", "-j", TorChain}})
	}
	for _, hook := range hooks {
		if err := h.runRule(hook); err != nil {
			h.ClearTransparentProxy(cfg)
			return err
		}
	}

	h.Logger.Println("[INFO] Transparent proxy rules applied successfully.")
	return nil
}

// ClearTransparentProxy unhooks and removes both chains and the automap route.
//
// Every step is best-effort: stop has to work after a failed start, after a
// successful one, and when nothing was ever applied. Refusing to continue past
// the first error would strand the caller with rules in place and no way to
// remove them. The order matters -- rules go before the route, so there is no
// moment where a redirect points at an address the kernel can no longer route.
func (h *CommandHandler) ClearTransparentProxy(cfg ProxyConfig) error {
	steps := []Rule{
		{Command: "iptables", Args: []string{"-t", "nat", "-D", "OUTPUT", "-j", TorChain}},
		{Command: "ip6tables", Args: []string{"-t", "nat", "-D", "OUTPUT", "-j", TorChain}},
		{Command: "iptables", Args: []string{"-t", "nat", "-F", TorChain}},
		{Command: "ip6tables", Args: []string{"-t", "nat", "-F", TorChain}},
		{Command: "iptables", Args: []string{"-t", "nat", "-X", TorChain}},
		{Command: "ip6tables", Args: []string{"-t", "nat", "-X", TorChain}},
		{Command: "ip", Args: virtualRouteArgs("del", cfg.VirtualNetIPv6)},
	}

	failures := 0
	for _, rule := range steps {
		if err := h.runRule(rule); err != nil {
			failures++
		}
	}

	if failures == len(steps) {
		h.Logger.Printf("[INFO] Nothing to remove; no %s chain or automap route was present.", TorChain)
		return nil
	}

	h.Logger.Println("[INFO] Transparent proxy rules cleared.")
	return nil
}
