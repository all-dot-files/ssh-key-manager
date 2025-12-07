package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/all-dot-files/ssh-key-manager/internal/storage/sqlite"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate-to-sqlite",
	Short: "Migrate configuration from YAML to SQLite",
	Long:  `Migrate all hosts, keys, and repositories from the current YAML configuration to a new SQLite database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configManager.Get()
		
		// Determine database path (same dir as config file)
		configDir := filepath.Dir(configManager.GetConfigPath())
		dbPath := filepath.Join(configDir, "skm.db")
		
		fmt.Printf("Migrating data to SQLite database: %s\n", dbPath)
		
		// Initialize SQLite store
		store, err := sqlite.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		defer store.Close()
		
		ctx := context.Background()
		
		// Migrate Hosts
		fmt.Printf("Migrating %d hosts...\n", len(cfg.Hosts))
		for _, host := range cfg.Hosts {
			if err := store.Host().Add(ctx, host); err != nil {
				fmt.Printf("⚠️  Failed to migrate host %s: %v\n", host.Host, err)
			}
		}
		
		// Migrate Keys
		fmt.Printf("Migrating %d keys...\n", len(cfg.Keys))
		for _, key := range cfg.Keys {
			if err := store.Key().Add(ctx, key); err != nil {
				fmt.Printf("⚠️  Failed to migrate key %s: %v\n", key.Name, err)
			}
		}
		
		// Migrate Repos
		fmt.Printf("Migrating %d repositories...\n", len(cfg.Repos))
		for _, repo := range cfg.Repos {
			if err := store.Repo().Add(ctx, repo); err != nil {
				fmt.Printf("⚠️  Failed to migrate repo %s: %v\n", repo.Path, err)
			}
		}
		
		fmt.Println("\n✅ Migration completed!")
		fmt.Println("To use the database, we will need to update SKM configuration (future logic).")
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
