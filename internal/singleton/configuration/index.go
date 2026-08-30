package configuration

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Configuration represents the application configuration.
type Configuration struct {
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Proxy     ProxySettings   `yaml:"proxy"`
}

// RateLimitConfig represents rate limit settings.
type RateLimitConfig struct {
	MinReadRate  uint `yaml:"min_read_rate"`
	MinWriteRate uint `yaml:"min_write_rate"`
}

// ProxySettings tunes the transparent proxy rules.
//
// Every field is optional and an absent one means "use the built-in default".
// That distinction matters: a configuration file written before these settings
// existed would otherwise yield empty networks, and the rules built from them
// would be malformed rather than merely unconfigured.
type ProxySettings struct {
	// VirtualNetIPv4 and VirtualNetIPv6 must match VirtualAddrNetworkIPv4 and
	// VirtualAddrNetworkIPv6 in torrc. A mismatch does not fail loudly: traffic
	// to resolved hosts simply stops being redirected, so connections keep
	// working while silently leaving Tor.
	VirtualNetIPv4 string `yaml:"virtual_net_ipv4"`
	VirtualNetIPv6 string `yaml:"virtual_net_ipv6"`

	// ExcludedNets and ExcludedNetsIPv6 are destinations that stay off Tor.
	// Hosts whose LAN falls outside the RFC1918 ranges need to say so here or
	// their local traffic is routed through Tor and fails.
	ExcludedNets     []string `yaml:"excluded_nets"`
	ExcludedNetsIPv6 []string `yaml:"excluded_nets_ipv6"`

	// EnableIPv6 is a pointer so that an absent key can be told apart from an
	// explicit false. Defaulting a missing key to false would quietly disable
	// IPv6 for everyone upgrading from a configuration written before it
	// existed, which is the opposite of the safe choice: Tor still hands out
	// virtual IPv6 addresses, and with no rules to catch them applications on
	// a dual-stack host would fail to connect.
	EnableIPv6 *bool `yaml:"enable_ipv6"`
}

var (
	instance *Configuration
	loadErr  error
	once     sync.Once
)

// GetConfig returns the singleton configuration instance.
func GetConfig() *Configuration {
	return instance
}

// LoadConfig reads and parses the configuration file. It is safe to call more
// than once; every caller sees the outcome of the first attempt.
func LoadConfig(configPath string) error {
	once.Do(func() {
		instance, loadErr = readConfig(configPath)
	})
	// loadErr is package scoped on purpose. It used to be a local variable set
	// inside once.Do, so the second and later callers were handed a nil error
	// no matter how the first attempt went -- reporting success while the
	// configuration was never loaded.
	return loadErr
}

func readConfig(configPath string) (*Configuration, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	config := &Configuration{}
	if err := yaml.NewDecoder(file).Decode(config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}
	return config, nil
}
