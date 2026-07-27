package cmd

import (
	"fmt"

	"github.com/Seicrypto/torcontroller/initializer"
	"github.com/Seicrypto/torcontroller/internal/controller"
	"github.com/spf13/cobra"
)

// InitCmd is the Cobra command for initializing all configurations.
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize all configurations for Torcontroller",
	Long: `The "init" command initializes all necessary configuration files
and sets up required system settings for Torcontroller. This includes
overwriting existing files with templates and generating a new password.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize the Initializer with embedded templates and a real command runner
		runner := &controller.RealCommandRunner{}
		fs := &initializer.RealFileSystem{}
		templateProvider := &initializer.EmbedFSWrapper{FS: initializer.Templates} // Wrap embed.FS
		initializer := initializer.NewInitializer(templateProvider, runner, fs)

		fmt.Println("[INFO] Initializing all configurations...")
		if err := initializer.InitializeAllConfig(); err != nil {
			fmt.Printf("[ERROR] Initialization failed: %v\n", err)
			return err
		}

		fmt.Println("[INFO] All configurations initialized successfully.")
		return nil
	},
}
