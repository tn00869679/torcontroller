package initializer

import (
	"fmt"

	"github.com/Seicrypto/torcontroller/internal/singleton/configuration"
	"gopkg.in/yaml.v3"
)

func (i *Initializer) VerifyConfigFile(path string) bool {
	info, err := i.FileSystem.Stat(path)
	if i.FileSystem.IsNotExist(err) {
		fmt.Printf("[ERROR] Configuration file not found: %s\n", path)
		return false
	} else if err != nil {
		fmt.Printf("Failed to check Configuration file: %v\n", err)
		return false
	}

	if info.IsDir() {
		fmt.Printf("[ERROR] Configuration path is a directory, not a file: %s\n", path)
		return false
	}

	content, err := i.FileSystem.ReadFile(path)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read configuration file: %v\n", err)
		return false
	}

	var cfg configuration.Configuration
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		fmt.Printf("[ERROR] Invalid configuration format: %v\n", err)
		return false
	}

	fmt.Println("[INFO] Configuration file is valid.")
	return true
}
