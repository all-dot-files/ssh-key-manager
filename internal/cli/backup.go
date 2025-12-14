package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/all-dot-files/ssh-key-manager/internal/backup"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage configuration backups",
	Long:  `Create, list, and restore backups of your SKM configuration and keys.`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		msg, _ := cmd.Flags().GetString("message")

		bm, err := backup.NewManager(configManager.GetConfigDir())
		if err != nil {
			return err
		}

		path, err := bm.Create(msg)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		fmt.Printf("✅ Backup created successfully: %s\n", path)
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		bm, err := backup.NewManager(configManager.GetConfigDir())
		if err != nil {
			return err
		}

		backups, err := bm.List()
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		if len(backups) == 0 {
			fmt.Println("No backups found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTIME\tSIZE")
		for _, b := range backups {
			fmt.Fprintf(w, "%s\t%s\t%s\n", b.Name, b.Timestamp.Format("2006-01-02 15:04:05"), formatSize(b.Size))
		}
		w.Flush()

		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [backup-file]",
	Short: "Restore configuration from a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		bm, err := backup.NewManager(configManager.GetConfigDir())
		if err != nil {
			return err
		}

		// If plain filename is provided, look in backup dir, else assume absolute path
		path := filename
		if filepath.Base(filename) == filename {
			// Try to find in backup dir
			potentialPath := filepath.Join(configManager.GetConfigDir(), backup.DefaultBackupDir, filename)
			if _, err := os.Stat(potentialPath); err == nil {
				path = potentialPath
			}
		}

		fmt.Printf("⚠️  Warning: This will overwrite your current configuration.\n")
		fmt.Printf("Restoring from: %s\n", path)

		// Simple confirmation (unless force flag?)
		// For now, let's just do it or require user to be careful.
		// Ideally we should ask for confirmation but keeping it simple for now.

		if err := bm.Restore(path); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}

		fmt.Println("✅ Restore completed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)

	backupCreateCmd.Flags().StringP("message", "m", "", "Optional message/tag for the backup")
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
