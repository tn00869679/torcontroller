package controller_test

import (
	"os"
	"strings"
	"testing"

	"github.com/tn00869679/torcontroller/internal/controller"
	"github.com/tn00869679/torcontroller/internal/singleton/configuration"
)

const torrcMatching = `ControlPort 9051
TransPort 127.0.0.1:9040
DNSPort 5353
AutomapHostsOnResolve 1
VirtualAddrNetworkIPv4 10.192.0.0/10
VirtualAddrNetworkIPv6 [fc00::]/7
`

func handlerWithTorrc(contents string) *controller.CommandHandler {
	return &controller.CommandHandler{
		Logger: NewMockLogger(),
		FileSystem: &MockFileSystem{
			Files: map[string]*MockFileInfo{
				"/etc/tor/torrc": {content: []byte(contents)},
			},
		},
	}
}

// torrc writes the IPv6 range as [fc00::]/7 while iptables takes fc00::/7.
// Treating those as different would reject a perfectly consistent setup.
func TestTorrcCheckAcceptsTheBracketedIPv6Spelling(t *testing.T) {
	handler := handlerWithTorrc(torrcMatching)

	if err := handler.VerifyTorrcMatchesProxyConfig(controller.DefaultProxyConfig("102"), "/etc/tor/torrc"); err != nil {
		t.Fatalf("expected the check to pass, got: %v", err)
	}
}

// The failure this guards against is silent: Tor hands out addresses from its
// range, the rules redirect a different one, and connections succeed while
// bypassing Tor entirely.
func TestTorrcCheckRejectsAMismatchedIPv4Range(t *testing.T) {
	handler := handlerWithTorrc(torrcMatching)
	cfg := controller.DefaultProxyConfig("102")
	cfg.VirtualNetIPv4 = "172.16.0.0/12"

	err := handler.VerifyTorrcMatchesProxyConfig(cfg, "/etc/tor/torrc")
	if err == nil {
		t.Fatal("expected a refusal when the ranges disagree")
	}
	if !strings.Contains(err.Error(), "VirtualAddrNetworkIPv4") {
		t.Errorf("the error should name the directive, got: %v", err)
	}
}

func TestTorrcCheckRejectsAMismatchedIPv6Range(t *testing.T) {
	handler := handlerWithTorrc(torrcMatching)
	cfg := controller.DefaultProxyConfig("102")
	cfg.VirtualNetIPv6 = "fd00::/8"

	if err := handler.VerifyTorrcMatchesProxyConfig(cfg, "/etc/tor/torrc"); err == nil {
		t.Fatal("expected a refusal when the IPv6 ranges disagree")
	}
}

// An IPv4-only run must not be blocked by an IPv6 directive it will never use.
func TestTorrcCheckIgnoresIPv6WhenItIsDisabled(t *testing.T) {
	handler := handlerWithTorrc(`VirtualAddrNetworkIPv4 10.192.0.0/10
`)
	cfg := controller.DefaultProxyConfig("102")
	cfg.EnableIPv6 = false

	if err := handler.VerifyTorrcMatchesProxyConfig(cfg, "/etc/tor/torrc"); err != nil {
		t.Fatalf("IPv6 directives should be irrelevant when IPv6 is off, got: %v", err)
	}
}

// A torrc without the directive at all means automap is not configured, so the
// rules would redirect a range Tor never hands out.
func TestTorrcCheckRejectsAMissingDirective(t *testing.T) {
	handler := handlerWithTorrc("ControlPort 9051\n")

	if err := handler.VerifyTorrcMatchesProxyConfig(controller.DefaultProxyConfig("102"), "/etc/tor/torrc"); err == nil {
		t.Fatal("expected a refusal when the directive is absent")
	}
}

// Commented-out directives are not in effect and must not satisfy the check.
func TestTorrcCheckIgnoresCommentedDirectives(t *testing.T) {
	handler := handlerWithTorrc(`#VirtualAddrNetworkIPv4 10.192.0.0/10
ControlPort 9051
`)

	if err := handler.VerifyTorrcMatchesProxyConfig(controller.DefaultProxyConfig("102"), "/etc/tor/torrc"); err == nil {
		t.Fatal("a commented directive must not count as configured")
	}
}

// A configuration file written before these keys existed decodes to zero
// values. Using them directly would build rules with empty networks.
func TestSettingsWithoutAProxyBlockFallBackToTheDefaults(t *testing.T) {
	settings := configuration.MockConfiguration(10, 5)

	cfg := controller.ProxyConfigFromSettings("102", settings)
	defaults := controller.DefaultProxyConfig("102")

	if cfg.VirtualNetIPv4 != defaults.VirtualNetIPv4 || cfg.VirtualNetIPv6 != defaults.VirtualNetIPv6 {
		t.Errorf("expected the default ranges, got %s and %s", cfg.VirtualNetIPv4, cfg.VirtualNetIPv6)
	}
	if len(cfg.ExcludedNets) != len(defaults.ExcludedNets) {
		t.Errorf("expected the default exclusions, got %v", cfg.ExcludedNets)
	}
	if !cfg.EnableIPv6 {
		t.Error("an absent enable_ipv6 must not disable IPv6: Tor still hands out virtual IPv6 addresses, and nothing would catch them")
	}
}

func TestSettingsOverrideTheDefaultsWhenPresent(t *testing.T) {
	disabled := false
	settings := &configuration.Configuration{
		Proxy: configuration.ProxySettings{
			VirtualNetIPv4: "172.16.0.0/12",
			VirtualNetIPv6: "fd12::/16",
			ExcludedNets:   []string{"198.51.100.0/24"},
			EnableIPv6:     &disabled,
		},
	}

	cfg := controller.ProxyConfigFromSettings("102", settings)

	if cfg.VirtualNetIPv4 != "172.16.0.0/12" || cfg.VirtualNetIPv6 != "fd12::/16" {
		t.Errorf("ranges not applied: %s %s", cfg.VirtualNetIPv4, cfg.VirtualNetIPv6)
	}
	if len(cfg.ExcludedNets) != 1 || cfg.ExcludedNets[0] != "198.51.100.0/24" {
		t.Errorf("exclusions not applied: %v", cfg.ExcludedNets)
	}
	if cfg.EnableIPv6 {
		t.Error("an explicit false must disable IPv6")
	}
	// The IPv6 exclusions were not set, so they keep their defaults.
	if len(cfg.ExcludedNetsIPv6) == 0 {
		t.Error("unset lists should keep their defaults rather than becoming empty")
	}
}

// The path was hardcoded to /var/lib/tor, which only worked because the
// shipped torrc overrides it there. Reading the value keeps the code and the
// configuration from drifting apart.
func TestCookiePathComesFromTorrc(t *testing.T) {
	handler := handlerWithTorrc("CookieAuthFile /var/lib/tor/control.authcookie\n")

	path, err := handler.CookieAuthPath("/etc/tor/torrc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/var/lib/tor/control.authcookie" {
		t.Errorf("expected the path from torrc, got %s", path)
	}
}

// Debian's stock torrc has no CookieAuthFile; its defaults file puts the
// cookie under /run/tor. Guessing /var/lib/tor there is what made switch and
// traffic fail with nothing pointing at the cause.
func TestCookiePathFallsBackToTheDebianDefault(t *testing.T) {
	handler := handlerWithTorrc("ControlPort 9051\nCookieAuthentication 1\n")

	path, err := handler.CookieAuthPath("/etc/tor/torrc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != controller.DefaultCookieAuthPath {
		t.Errorf("expected %s, got %s", controller.DefaultCookieAuthPath, path)
	}
}

// A shipped hash would be the same secret on every installation, so anyone
// knowing the plaintext could drive the control port of any machine running
// this package.
func TestShippedTorrcCarriesNoControlPassword(t *testing.T) {
	contents, err := os.ReadFile("../../initializer/templates/tor/torrc")
	if err != nil {
		t.Fatalf("could not read the shipped torrc: %v", err)
	}

	for _, line := range strings.Split(string(contents), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "HashedControlPassword") {
			t.Errorf("the shipped torrc must not carry a password hash, found: %s", line)
		}
	}
}

// Every caller has to see the outcome of the first attempt. Returning nil to
// later callers reported success while nothing had been loaded.
func TestLoadConfigReportsTheFirstFailureToEveryCaller(t *testing.T) {
	first := configuration.LoadConfig("/nonexistent/torcontroller.yml")
	if first == nil {
		t.Fatal("expected an error for a missing file")
	}

	second := configuration.LoadConfig("/nonexistent/torcontroller.yml")
	if second == nil {
		t.Error("the second caller was told the load succeeded when it never did")
	}
}
