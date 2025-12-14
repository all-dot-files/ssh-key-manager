package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var keyImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import existing SSH keys and host bindings from a directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configManager == nil {
			return fmt.Errorf("config manager not initialized")
		}
		dir, _ := cmd.Flags().GetString("dir")
		detectHosts, _ := cmd.Flags().GetBool("detect-hosts")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		aliasPrefix, _ := cmd.Flags().GetString("alias-prefix")

		planner := NewImportPlanner(configManager, dir, aliasPrefix, dryRun)
		keys, err := planner.DiscoverKeys()
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return fmt.Errorf("no keys found in %s", dir)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Found %d key(s) in %s\n", len(keys), dir)
		for _, k := range keys {
			fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", k.Alias, k.Status)
		}

		if detectHosts {
			bindings, err := planner.DetectBindings()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Detected %d host binding(s)\n", len(bindings))
			for _, b := range bindings {
				conflict := ""
				if b.Conflict {
					conflict = " (conflict)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "- Host %s -> %s%s\n", b.HostPattern, b.TargetAlias, conflict)
			}
		}

		for _, w := range planner.Warnings() {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠️  %s\n", w)
		}

		if dryRun {
			fmt.Fprintln(cmd.OutOrStdout(), "Dry run complete. No files written.")
			return nil
		}

		if err := planner.Apply(); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Import complete.")
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyImportCmd)
	keyImportCmd.Flags().String("dir", filepath.Join("~", ".ssh"), "Directory containing SSH keys and config")
	keyImportCmd.Flags().Bool("detect-hosts", true, "Parse ssh config to auto-bind hosts to imported keys")
	keyImportCmd.Flags().Bool("dry-run", false, "Preview imports and bindings without writing")
	keyImportCmd.Flags().String("alias-prefix", "", "Prefix to apply to imported key aliases")
}
