package controller

import (
	"fmt"
	"strings"

	"github.com/tn00869679/torcontroller/internal/singleton/configuration"
)

// TorrcPath is where the shipped torrc lives.
const TorrcPath = "/etc/tor/torrc"

// ProxyConfigFromSettings builds the rule configuration from the file, falling
// back to the defaults for anything it does not set.
//
// Absent is not the same as empty here. A configuration written before these
// keys existed decodes to zero values, and rules built from those would carry
// an empty network -- malformed rather than merely unconfigured.
func ProxyConfigFromSettings(torUID string, settings *configuration.Configuration) ProxyConfig {
	cfg := DefaultProxyConfig(torUID)
	if settings == nil {
		return cfg
	}

	if settings.Proxy.VirtualNetIPv4 != "" {
		cfg.VirtualNetIPv4 = settings.Proxy.VirtualNetIPv4
	}
	if settings.Proxy.VirtualNetIPv6 != "" {
		cfg.VirtualNetIPv6 = settings.Proxy.VirtualNetIPv6
	}
	if len(settings.Proxy.ExcludedNets) > 0 {
		cfg.ExcludedNets = settings.Proxy.ExcludedNets
	}
	if len(settings.Proxy.ExcludedNetsIPv6) > 0 {
		cfg.ExcludedNetsIPv6 = settings.Proxy.ExcludedNetsIPv6
	}
	if settings.Proxy.EnableIPv6 != nil {
		cfg.EnableIPv6 = *settings.Proxy.EnableIPv6
	}
	return cfg
}

// VerifyTorrcMatchesProxyConfig checks that the automap ranges in torrc are the
// ones the rules will redirect.
//
// A mismatch is the worst kind of failure this feature can have. Tor hands out
// addresses from its range, the rules redirect a different one, so connections
// to every resolved host succeed while quietly bypassing Tor. Nothing errors,
// nothing logs, and the user believes they are anonymous. Refusing to start is
// the only honest response.
func (h *CommandHandler) VerifyTorrcMatchesProxyConfig(cfg ProxyConfig, torrcPath string) error {
	contents, err := h.FileSystem.ReadFile(torrcPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", torrcPath, err)
	}
	directives := parseTorrcDirectives(string(contents))

	expected := map[string]string{"VirtualAddrNetworkIPv4": cfg.VirtualNetIPv4}
	if cfg.EnableIPv6 {
		expected["VirtualAddrNetworkIPv6"] = cfg.VirtualNetIPv6
	}

	var problems []string
	for directive, want := range expected {
		got, present := directives[directive]
		if !present {
			problems = append(problems, fmt.Sprintf("%s is missing from %s (expected %s)", directive, torrcPath, want))
			continue
		}
		if normalizeNetwork(got) != normalizeNetwork(want) {
			problems = append(problems, fmt.Sprintf("%s is %s in %s but %s in the configuration", directive, got, torrcPath, want))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf(
			"torrc and the proxy configuration disagree: %s. Traffic to resolved hosts would leave Tor without any error being reported, so the rules will not be installed",
			strings.Join(problems, "; "),
		)
	}

	h.Logger.Println("[INFO] torrc automap ranges match the proxy configuration.")
	return nil
}

// DefaultCookieAuthPath is where Debian's tor-service-defaults-torrc puts the
// control cookie when torrc does not say otherwise.
const DefaultCookieAuthPath = "/run/tor/control.authcookie"

// CookieAuthPath reports where Tor writes the control authentication cookie.
//
// The path used to be hardcoded to /var/lib/tor/control.authcookie, which
// happened to work only because the shipped torrc overrides it there. Anyone
// running Debian's stock torrc, where the cookie lives under /run/tor, found
// switch and traffic failing at the read with nothing pointing at the cause.
// Reading the value keeps the code and the configuration from drifting apart.
func (h *CommandHandler) CookieAuthPath(torrcPath string) (string, error) {
	contents, err := h.FileSystem.ReadFile(torrcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", torrcPath, err)
	}

	if path, ok := parseTorrcDirectives(string(contents))["CookieAuthFile"]; ok {
		return path, nil
	}
	return DefaultCookieAuthPath, nil
}

// parseTorrcDirectives collects the last value seen for each directive, which
// is the one Tor itself honours for these settings.
func parseTorrcDirectives(contents string) map[string]string {
	directives := make(map[string]string)
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		directives[parts[0]] = parts[1]
	}
	return directives
}

// normalizeNetwork makes the two spellings of an IPv6 range comparable: torrc
// writes [fc00::]/7 while iptables takes fc00::/7.
func normalizeNetwork(network string) string {
	return strings.NewReplacer("[", "", "]", "").Replace(strings.TrimSpace(network))
}
