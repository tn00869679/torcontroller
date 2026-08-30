package cmd

import (
	"context"
	"fmt"

	"github.com/tn00869679/torcontroller/internal/controller"
	"github.com/tn00869679/torcontroller/internal/singleton/configuration"
	"github.com/tn00869679/torcontroller/internal/singleton/logger"
	"github.com/spf13/cobra"
)

var socketPath = "/tmp/torcontroller.sock"

// Define a private type to avoid conflicts
type contextKey string

const HandlerKey = contextKey("handler")
const FileSystem = contextKey("fileSystem")
const Logger = contextKey("logger")

// commandRunsWithoutConfig reports whether a subcommand must work before
// /etc/torcontroller/torcontroller.yml exists.
func commandRunsWithoutConfig(name string) bool {
	switch name {
	case "version", "migrate":
		return true
	default:
		return false
	}
}

// Root Command
var rootCmd = &cobra.Command{
	Use:   "torcontroller",
	Short: "Tor Controller CLI",
	Long:  "A CLI to control Tor and Privoxy services.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		// Initialize configuration.
		//
		// Two commands are exempt because they have to work on a machine whose
		// configuration is missing or half-written. `version` answering "open
		// /etc/torcontroller/torcontroller.yml: no such file" to someone trying
		// to find out what they have installed is useless, and `migrate` exists
		// precisely to repair an installation that is not yet in order.
		if !commandRunsWithoutConfig(cmd.Name()) {
			configurationPath := "/etc/torcontroller/torcontroller.yml"
			if err := configuration.LoadConfig(configurationPath); err != nil {
				return fmt.Errorf("failed to load configuration: %v", err)
			}
		}

		// Initialize the SocketInteractionHandler
		handler := &SocketInteractionHandler{
			Adapter: &UnixSocketAdapter{SocketPath: socketPath},
		}

		// Initialize singleton logger
		logger := logger.GetLogger()

		// Initialize the real file system
		fs := &controller.RealFileSystem{}

		// Pass these instances into the context
		ctx := cmd.Context()
		ctx = context.WithValue(ctx, HandlerKey, handler)
		ctx = context.WithValue(ctx, FileSystem, fs)
		ctx = context.WithValue(ctx, Logger, logger)
		cmd.SetContext(ctx)

		return nil
	},
}

var pidFile = "/tmp/torcontroller.pid"

var log *logger.Logger

// Initialize Root Command
func InitCommands() *cobra.Command {
	// Initialization Log
	log = logger.GetLogger()

	rootCmd.AddCommand(
		VersionCmd,
		CheckCmd,
		InitCmd,
		StartCmd,
		StartBackgroundCmd,
		TrafficCmd,
		SwitchCmd,
		StopCmd,
		NewPasswordCmd,
		MigrateCmd,
	)

	CheckCmd.Flags().BoolVarP(&fixFlag, "fix", "f", false, "Fix missing or incorrect results")

	return rootCmd
}
