package cmd

import (
	"fmt"

	"github.com/tn00869679/torcontroller/initializer"
	"github.com/tn00869679/torcontroller/internal/controller"
	"github.com/spf13/cobra"
)

// MigrateCmd brings an installation made by an earlier version up to date.
//
// It is hidden because postinst is what runs it. Exposing it in help would
// invite people to run it expecting something to happen, when on a current
// installation it correctly does nothing at all.
var MigrateCmd = &cobra.Command{
	Use:    "migrate",
	Short:  "Bring configuration from an earlier version up to date",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		runner := &controller.RealCommandRunner{}
		fs := &initializer.RealFileSystem{}
		templates := &initializer.EmbedFSWrapper{FS: initializer.Templates}
		init := initializer.NewInitializer(templates, runner, fs)

		result, err := init.Migrate()
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		out := cmd.OutOrStdout()
		if result.AlreadyUpToDate {
			fmt.Fprintln(out, "[INFO] Configuration is already up to date.")
			return nil
		}

		if len(result.AddedDirectives) > 0 {
			fmt.Fprintf(out, "[INFO] Added to %s: %v\n", initializer.MigrateTorrcPath, result.AddedDirectives)
			fmt.Fprintln(out, "[INFO] Restart tor for them to take effect: systemctl restart tor")
		}
		if result.RemovedPassword {
			fmt.Fprintln(out, "[INFO] Removed the control password that earlier versions shipped.")
			fmt.Fprintln(out, "[INFO] It was identical on every installation. Cookie authentication is used instead.")
		}
		if result.MovedLegacyUnit {
			fmt.Fprintf(out, "[INFO] Moved %s to %s.\n", initializer.LegacyUnitPath, initializer.LegacyUnitBackup)
			fmt.Fprintln(out, "[INFO] Debian's own tor unit applies now; it runs Tor as debian-tor,")
			fmt.Fprintln(out, "[INFO] which the redirection rules depend on.")
		}
		return nil
	},
}
