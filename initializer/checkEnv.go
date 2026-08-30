package initializer

import (
	"embed"
	"fmt"

	"github.com/tn00869679/torcontroller/internal/controller"
)

//go:embed templates/*
var Templates embed.FS

// CheckEnvironment validates and fixes the environment based on configuration.
func CheckEnvironment(fix bool) {
	fmt.Println("Environment Check Report:")

	// Initialize the Initializer with embedded templates and a real command runner
	runner := &controller.RealCommandRunner{}
	fs := &RealFileSystem{}
	templateProvider := &EmbedFSWrapper{FS: Templates} // Wrap embed.FS
	initializer := NewInitializer(templateProvider, runner, fs)

	configurationPath := "/etc/torcontroller/torcontroller.yml"
	// Configuration toncontroller.yml check
	if initializer.VerifyConfigFile(configurationPath) {
		fmt.Println("- Torcontroller Cofiguration File [OK]")
	} else {
		fmt.Println("- Torcontroller Cofiguration File [Invalid]")
		if fix {
			fmt.Println("  -> Attempting to place Torcontroller Cofiguration File...")
			if err := initializer.PlaceTorcontrollerYamlFile(configurationPath); err != nil {
				fmt.Printf("  [ERROR] Failed to place Torcontroller Cofiguration File: %v\n", err)
			} else {
				fmt.Println("  [INFO] Torcontroller Cofiguration File placed successfully.")
			}
		}
	}

	// Sudoer File Check
	if initializer.SudoersFileVerify() {
		fmt.Println("- Sudoers File [OK]")
	} else {
		fmt.Println("- Sudoers File [MISSING]")
		if fix {
			fmt.Println("  -> Attempting to place Sudoers File...")
			if err := initializer.PlaceSudoersFile(); err != nil {
				fmt.Printf("  [ERROR] Failed to place Sudoers File: %v\n", err)
			} else {
				fmt.Println("  [INFO] Sudoers File placed successfully.")
			}
		}
	}

	// Tor Service File Check. The unit belongs to Debian's tor package:
	// it drops privileges to debian-tor and creates /run/tor, and the
	// transparent proxy rules depend on both. Writing our own over it is
	// what left Tor running as root, unable to open its data directory.
	if initializer.CheckTorService() {
		fmt.Println("- Tor Service [OK]")
	} else {
		fmt.Println("- Tor Service [MISSING]")
		fmt.Println("  -> Reinstall it with: apt-get install --reinstall tor")
	}

	// Privoxy Service File Check
	if initializer.CheckPrivoxyService() {
		fmt.Println("- Privoxy Service [OK]")
	} else {
		fmt.Println("- Privoxy Service [NOT INSTALLED]")
		fmt.Println("  -> Optional. Traffic goes to Tor directly; install privoxy")
		fmt.Println("     only to filter it, then set http_proxy=127.0.0.1:8118")
		if fix {
			fmt.Println("  -> Attempting to place Privoxy Service...")
			if err := initializer.PlacePrivoxyServiceFile(); err != nil {
				fmt.Printf("  [ERROR] Failed to place Privoxy Service: %v\n", err)
			} else {
				fmt.Println("  [INFO] Privoxy Service placed successfully.")
			}
		}
	}

	// Torrc File Check
	if initializer.VerifyTorrcConfig() {
		fmt.Println("- Torrc config [OK]")
	} else {
		fmt.Println("- Torrc config [MISSING]")
		if fix {
			fmt.Println("  -> Attempting to place Torrc configuration...")
			if err := initializer.PlaceTorrcConfig(); err != nil {
				fmt.Printf("  [ERROR] Failed to place Torrc configuration: %v\n", err)
			} else {
				fmt.Println("  [INFO] Torrc configuration placed successfully.")
			}
		}
	}
}
