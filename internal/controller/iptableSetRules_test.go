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

// The chain must be complete before OUTPUT jumps to it. Hooking it early would
// expose a window where some traffic is redirected and the rest is not.
func TestChainIsPopulatedBeforeItIsHookedIntoOutput(t *testing.T) {
	runner := &recordingRunner{}
	handler := newTestHandler(runner)

	if err := handler.ApplyTransparentProxy(controller.DefaultProxyConfig("102")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hook := runner.indexOf("-A OUTPUT -j " + controller.TorChain)
	if hook == -1 {
		t.Fatal("OUTPUT was never hooked to the chain")
	}
	if hook != len(runner.calls)-1 {
		t.Errorf("the OUTPUT jump must be the final command, was %d of %d", hook, len(runner.calls))
	}
	if catchAll := runner.indexOf("-p tcp -j REDIRECT"); catchAll > hook {
		t.Errorf("catch-all rule added at %d, after the jump at %d", catchAll, hook)
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

	if err := handler.ClearTransparentProxy(); err != nil {
		t.Fatalf("clear should tolerate a missing hook, got: %v", err)
	}
	if runner.indexOf("-F "+controller.TorChain) == -1 {
		t.Error("expected the flush to run even though unhooking failed")
	}
	if runner.indexOf("-X "+controller.TorChain) == -1 {
		t.Error("expected the chain removal to run even though unhooking failed")
	}
}
