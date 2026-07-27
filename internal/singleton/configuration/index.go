package configuration

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Configuration struct {
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig represents rate limit settings.
type RateLimitConfig struct {
	MinReadRate  uint `yaml:"min_read_rate"`
	MinWriteRate uint `yaml:"min_write_rate"`
}

var (
	instance *Configuration
	once     sync.Once
)

// GetConfig returns the singleton configuration instance.
func GetConfig() *Configuration {
	return instance
}

// LoadConfig reads and parses the configuration file.
func LoadConfig(configPath string) error {
	var loadErr error // Local variable for error tracking

	once.Do(func() {
		file, err := os.Open(configPath)
		if err != nil {
			loadErr = fmt.Errorf("failed to open configuration file: %v", err)
			return
		}
		defer file.Close()

		config := &Configuration{}
		decoder := yaml.NewDecoder(file)
		if err = decoder.Decode(config); err != nil {
			loadErr = fmt.Errorf("failed to parse configuration file: %v", err)
			return
		}

		instance = config
	})

	return loadErr
}
