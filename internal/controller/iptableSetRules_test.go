package controller_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/tn00869679/torcontroller/internal/controller"
)

// recordingRunner keeps the order commands were issued in. The shared
// MockCommandRunner is map-backed and cannot answer "what ran before what",
// which is the property most of these tests exist to protect.
type recordingRunner struct {
	calls   []string
	failOn  string // substring of the command that should return an error
	callErr error
}

func (r *recordingRunner) Run(name string, args ...string) (string, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, command)
	if r.failOn != "" && strings.Contains(command, r.failOn) {
		if r.callErr == nil {
			r.callErr = errors.New("simulated failure")
		}
		return "", r.callErr
	}
	return "", nil
}

func (r *recordingRunner) indexOf(substring string) int {
	for i, call := range r.calls {
		if strings.Contains(call, substring) {
			return i
		}
	}
	return -1
}

func newTestHandler(runner controller.CommandRunner) *controller.CommandHandler {
	return &controller.CommandHandler{
		Logger:        NewMockLogger(),
		CommandRunner: runner,
	}
}

func ruleStrings(rules []controller.Rule) []string {
	out := make([]string, 0, len(rules))
	for _, rule := range rules {
		out = append(out, strings.Join(append([]string{rule.Command}, rule.Args...), " "))
	}
	return out
}

func indexOfRule(rules []controller.Rule, substring string) int {
	for i, rule := range ruleStrings(rules) {
		if strings.Contains(rule, substring) {
			return i
		}
	}
	return -1
}

// Tor reaches the network by making ordinary outbound connections. If those
// are redirected too, Tor is pointed at itself and the host loses the network
// entirely -- the failure that made the old 80/443 rules unusable.
func TestTorsOwnTrafficIsExemptedBeforeAnythingIsRedirected(t *testing.T) {
	rules := controller.BuildIPv4Rules(controller.DefaultProxyConfig("102"))

	exemption := indexOfRule(rules, "--uid-owner 102 -j RETURN")
	if exemption != 0 {
		t.Fatalf("Tor's exemption must be the first rule, found at index %d in:\n%s",
			exemption, strings.Join(ruleStrings(rules), "\n"))
	}
}

// The automap range sits inside 10.0.0.0/8. If the LAN exemption is consulted
// first, every hostname resolved through Tor resolves to an address that then
// bypasses Tor -- silently, because connections still succeed.
func TestVirtualAddressesAreRedirectedBeforeThePrivateNetworkExemption(t *testing.T) {
	cfg := controller.DefaultProxyConfig("102")
	rules := controller.BuildIPv4Rules(cfg)

	redirect := indexOfRule(rules, "-d "+cfg.VirtualNetIPv4+" -j REDIRECT")
	exemption := indexOfRule(rules, "-d 10.0.0.0/8 -j RETURN")

	if redirect == -1 || exemption == -1 {
		t.Fatalf("expected both rules to exist, got redirect=%d exemption=%d", redirect, exemption)
	}
	if redirect > exemption {
		t.Errorf("virtual range %s is redirected at index %d, after the 10.0.0.0/8 exemption at %d; traffic to resolved hosts would leave Tor",
			cfg.VirtualNetIPv4, redirect, exemption)
	}
}

// Resolvers commonly live on 127.0.0.53. Exempting loopback before catching
// DNS would let every query reach the network in clear text, exposing the
// browsing history the rest of this feature exists to hide.
func TestDNSIsRedirectedBeforeTheLoopbackExemption(t *testing.T) {
	rules := controller.BuildIPv4Rules(controller.DefaultProxyConfig("102"))

	dns := indexOfRule(rules, "--dport 53 -j REDIRECT")
	loopback := indexOfRule(rules, "-d 127.0.0.0/8 -j RETURN")

	if dns == -1 || loopback == -1 {
		t.Fatalf("expected both rules to exist, got dns=%d loopback=%d", dns, loopback)
	}
	if dns > loopback {
		t.Errorf("DNS redirect at index %d comes after the loopback exemption at %d; queries to a local resolver would leak", dns, loopback)
	}
}

// Redirecting all TCP rather than a port list is what carries protocols other
// than HTTP through Tor. It has to be last so the exemptions above still win.
func TestEveryRemainingTCPConnectionIsRedirectedLast(t *testing.T) {
	rules := controller.BuildIPv4Rules(controller.DefaultProxyConfig("102"))
	last := ruleStrings(rules)[len(rules)-1]

	if !strings.HasSuffix(last, "-p tcp -j REDIRECT --to-ports "+controller.TransPortIPv4) {
		t.Errorf("expected the catch-all TCP redirect last, got: %s", last)
	}
	for _, rule := range ruleStrings(rules) {
		if strings.Contains(rule, "--dport 80") || strings.Contains(rule, "--dport 443") {
			t.Errorf("rules should no longer single out web ports, found: %s", rule)
		}
	}
}

// Both chains must be complete before either OUTPUT jump is added. Hooking one
// early would expose a window where some traffic is redirected and the rest is
// not.
func TestChainsArePopulatedBeforeEitherIsHookedIntoOutput(t *testing.T) {
	runner := &recordingRunner{}
	handler := newTestHandler(runner)

	if err := handler.ApplyTransparentProxy(controller.DefaultProxyConfig("102")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v4Hook := runner.indexOf("iptables -t nat -A OUTPUT -j " + controller.TorChain)
	v6Hook := runner.indexOf("ip6tables -t nat -A OUTPUT -j " + controller.TorChain)
	if v4Hook == -1 || v6Hook == -1 {
		t.Fatalf("both jumps must be installed, got v4=%d v6=%d in:\n%s",
			v4Hook, v6Hook, strings.Join(runner.calls, "\n"))
	}

	// The two jumps are the final commands; nothing else may run after them.
	lastTwo := len(runner.calls) - 2
	if v4Hook < lastTwo || v6Hook < lastTwo {
		t.Errorf("the OUTPUT jumps must be the last two commands, found at %d and %d of %d",
			v4Hook, v6Hook, len(runner.calls))
	}
}

// A partially applied rule set is worse than none: it can redirect some
// traffic to a chain that never got its exemptions. Failure must leave the
// host exactly as it was.
func TestAFailedRuleRemovesTheChainAndNeverHooksOutput(t *testing.T) {
	runner := &recordingRunner{failOn: "--dport 53"}
	handler := newTestHandler(runner)

	err := handler.ApplyTransparentProxy(controller.DefaultProxyConfig("102"))
	if err == nil {
		t.Fatal("expected an error when a rule fails")
	}

	if hook := runner.indexOf("-A OUTPUT -j " + controller.TorChain); hook != -1 {
		t.Errorf("OUTPUT must not be hooked after a failure, but was at index %d", hook)
	}
	if teardown := runner.indexOf("-X " + controller.TorChain); teardown == -1 {
		t.Error("the chain should have been removed after the failure")
	}
}

// Without Tor's uid the exemption cannot be written, and the catch-all would
// redirect Tor into itself. Refusing is the only safe response.
func TestApplyRefusesWithoutTorsUID(t *testing.T) {
	runner := &recordingRunner{}
	handler := newTestHandler(runner)

	cfg := controller.DefaultProxyConfig("")
	if err := handler.ApplyTransparentProxy(cfg); err == nil {
		t.Fatal("expected a refusal when Tor's uid is unknown")
	}
	if len(runner.calls) != 0 {
		t.Errorf("no iptables command should have run, got: %v", runner.calls)
	}
}

// stop has to work after a failed start, after a successful one, and when
// nothing was ever applied. Bailing on the first error would strand rules in
// place with no way to remove them.
func TestClearContinuesWhenPartsOfTheChainAreAlreadyGone(t *testing.T) {
	runner := &recordingRunner{failOn: "-D OUTPUT"}
	handler := newTestHandler(runner)

	if err := handler.ClearTransparentProxy(controller.DefaultProxyConfig("")); err != nil {
		t.Fatalf("clear should tolerate a missing hook, got: %v", err)
	}
	if runner.indexOf("-F "+controller.TorChain) == -1 {
		t.Error("expected the flush to run even though unhooking failed")
	}
	if runner.indexOf("-X "+controller.TorChain) == -1 {
		t.Error("expected the chain removal to run even though unhooking failed")
	}
}

// The IPv6 half is inert without its route: a connection to a virtual address
// dies in the kernel's routing lookup, which runs before the nat chain, so no
// redirect rule ever sees the packet.
func TestVirtualRouteIsInstalledBeforeTheIPv6Rules(t *testing.T) {
	runner := &recordingRunner{}
	handler := newTestHandler(runner)
	cfg := controller.DefaultProxyConfig("102")

	if err := handler.ApplyTransparentProxy(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	route := runner.indexOf("ip -6 route add local " + cfg.VirtualNetIPv6)
	rule := runner.indexOf("ip6tables -t nat -A " + controller.TorChain)
	if route == -1 {
		t.Fatalf("the automap route was never added:\n%s", strings.Join(runner.calls, "\n"))
	}
	if route > rule {
		t.Errorf("route added at %d, after the first IPv6 rule at %d; the rules would have nothing to match", route, rule)
	}
}

// A route left behind by an interrupted run would make the add fail or stack a
// duplicate, so start replaces rather than appends.
func TestVirtualRouteIsReplacedNotAppended(t *testing.T) {
	runner := &recordingRunner{}
	handler := newTestHandler(runner)
	cfg := controller.DefaultProxyConfig("102")

	if err := handler.ApplyTransparentProxy(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	del := runner.indexOf("ip -6 route del local " + cfg.VirtualNetIPv6)
	add := runner.indexOf("ip -6 route add local " + cfg.VirtualNetIPv6)
	if del == -1 || del > add {
		t.Errorf("expected a delete before the add, got del=%d add=%d", del, add)
	}
}

// Leaving the route behind marks the whole range local, which makes real
// unique-local addresses unreachable. Teardown has to remove it, and only
// after the rules that depend on it are gone.
func TestClearRemovesTheVirtualRouteAfterTheRules(t *testing.T) {
	runner := &recordingRunner{}
	handler := newTestHandler(runner)
	cfg := controller.DefaultProxyConfig("")

	if err := handler.ClearTransparentProxy(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	route := runner.indexOf("ip -6 route del local " + cfg.VirtualNetIPv6)
	lastRule := runner.indexOf("ip6tables -t nat -X " + controller.TorChain)
	if route == -1 {
		t.Fatalf("the automap route was never removed:\n%s", strings.Join(runner.calls, "\n"))
	}
	if route < lastRule {
		t.Errorf("route removed at %d, before the rules at %d; redirects would briefly point at an unroutable address", route, lastRule)
	}
}

// Turning IPv6 off has to skip the route as well as the rules. Installing the
// route without the rules would strand the range as local with nothing serving
// it.
func TestDisablingIPv6SkipsBothItsRulesAndItsRoute(t *testing.T) {
	runner := &recordingRunner{}
	handler := newTestHandler(runner)
	cfg := controller.DefaultProxyConfig("102")
	cfg.EnableIPv6 = false

	if err := handler.ApplyTransparentProxy(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, forbidden := range []string{"ip6tables", "ip -6 route"} {
		if index := runner.indexOf(forbidden); index != -1 {
			t.Errorf("with IPv6 disabled, %q should not run, found at %d: %s", forbidden, index, runner.calls[index])
		}
	}
	if runner.indexOf("iptables -t nat -A OUTPUT -j "+controller.TorChain) == -1 {
		t.Error("the IPv4 half should still be installed")
	}
}

// Same ordering contract as IPv4: Tor's own traffic must escape redirection
// before any rule can catch it.
func TestIPv6RulesExemptTorFirstAndCatchEverythingLast(t *testing.T) {
	cfg := controller.DefaultProxyConfig("102")
	rules := controller.BuildIPv6Rules(cfg)

	if index := indexOfRule(rules, "--uid-owner 102 -j RETURN"); index != 0 {
		t.Errorf("Tor's exemption must come first, found at %d", index)
	}
	last := ruleStrings(rules)[len(rules)-1]
	if !strings.HasSuffix(last, "-p tcp -j REDIRECT --to-ports "+controller.TransPortIPv6) {
		t.Errorf("expected the catch-all last, got: %s", last)
	}
	if virtual := indexOfRule(rules, "-d "+cfg.VirtualNetIPv6+" -j REDIRECT"); virtual == -1 {
		t.Error("the automap range must be redirected explicitly")
	}
}
