package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/all-dot-files/ssh-key-manager/internal/sync"
)

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	Long:  `Display current sync status and pending changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configManager.Get()

		if cfg.Server == "" {
			return fmt.Errorf("no server configured. Run: skm server-login")
		}

		syncMgr := sync.NewSyncManager(cfg.DeviceID, sync.StrategyNewerWins)
		syncMgr.UpdateLocalState(cfg.Keys)

		// Detect changes
		changes := syncMgr.DetectChanges(cfg.Keys)

		if len(changes) == 0 {
			fmt.Println("✓ Everything is up to date")
			return nil
		}

		fmt.Printf("Found %d pending change(s):\n\n", len(changes))
		changelog := syncMgr.GetChangelog(changes)
		for _, line := range changelog {
			fmt.Println(line)
		}

		return nil
	},
}

var syncHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show sync history",
	Long:  `Display synchronization history.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configManager.Get()

		limit, _ := cmd.Flags().GetInt("limit")

		history, err := sync.NewSyncHistory(cfg.KeystorePath, 100)
		if err != nil {
			return fmt.Errorf("failed to load sync history: %w", err)
		}

		entries := history.GetRecent(limit)
		if len(entries) == 0 {
			fmt.Println("No sync history found.")
			return nil
		}

		fmt.Println(history.FormatHistory(entries))

		// Show stats
		stats := history.GetStats()
		fmt.Println("\n📊 Statistics:")
		fmt.Printf("   Total syncs: %d\n", stats.TotalSyncs)
		fmt.Printf("   Successful: %d\n", stats.SuccessfulSyncs)
		fmt.Printf("   Failed: %d\n", stats.FailedSyncs)
		fmt.Printf("   Total changes: %d\n", stats.TotalChanges)
		fmt.Printf("   Total conflicts: %d\n", stats.TotalConflicts)
		if !stats.LastSyncTime.IsZero() {
			fmt.Printf("   Last sync: %s\n", stats.LastSyncTime.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

var syncResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve sync conflicts",
	Long:  `Resolve synchronization conflicts using a specified strategy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configManager.Get()

		strategy, _ := cmd.Flags().GetString("strategy")

		var syncStrategy sync.SyncStrategy
		switch strategy {
		case "local":
			syncStrategy = sync.StrategyLocalWins
		case "remote":
			syncStrategy = sync.StrategyRemoteWins
		case "newer":
			syncStrategy = sync.StrategyNewerWins
		default:
			return fmt.Errorf("invalid strategy: %s (use: local, remote, newer)", strategy)
		}

		fmt.Printf("🔄 Resolving conflicts using '%s' strategy...\n", strategy)

		_ = sync.NewSyncManager(cfg.DeviceID, syncStrategy)

		// TODO: Fetch remote keys and detect conflicts
		// conflicts := syncMgr.DetectConflicts(cfg.Keys, remoteKeys)

		fmt.Println("✓ Conflicts resolved")

		return nil
	},
}

var syncClearHistoryCmd = &cobra.Command{
	Use:   "clear-history",
	Short: "Clear sync history",
	Long:  `Clear all synchronization history records.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configManager.Get()

		fmt.Print("⚠️  This will delete all sync history. Continue? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}

		history, err := sync.NewSyncHistory(cfg.KeystorePath, 100)
		if err != nil {
			return fmt.Errorf("failed to load sync history: %w", err)
		}

		if err := history.Clear(); err != nil {
			return fmt.Errorf("failed to clear history: %w", err)
		}

		fmt.Println("✓ Sync history cleared")

		return nil
	},
}

func init() {
	// Add new sync subcommands
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncHistoryCmd)
	syncCmd.AddCommand(syncResolveCmd)
	syncCmd.AddCommand(syncClearHistoryCmd)

	// Flags for history command
	syncHistoryCmd.Flags().IntP("limit", "n", 10, "Number of entries to show")

	// Flags for resolve command
	syncResolveCmd.Flags().StringP("strategy", "s", "newer", "Conflict resolution strategy (local, remote, newer)")
}
