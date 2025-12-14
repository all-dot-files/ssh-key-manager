package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/all-dot-files/ssh-key-manager/internal/keystore"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/pkg/errors"
)

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Manage SSH hosts",
	Long:  `Add, list, and manage SSH host configurations.`,
}

var hostAddCmd = &cobra.Command{
	Use:   "add <hostname>",
	Short: "Add a new SSH host configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]
		user, _ := cmd.Flags().GetString("user")
		keyName, _ := cmd.Flags().GetString("key")
		port, _ := cmd.Flags().GetInt("port")
		actualHost, _ := cmd.Flags().GetString("hostname")
		autoCreate, _ := cmd.Flags().GetBool("auto-create-key")

		if user == "" {
			return fmt.Errorf("user is required")
		}

		// Auto-create key if requested and key doesn't exist
		if keyName == "" || autoCreate {
			if keyName == "" {
				keyName = fmt.Sprintf("%s-key", hostname)
			}

			// Check if key exists
			if _, err := configManager.GetKey(keyName); err != nil {
				fmt.Printf("Key '%s' not found. Creating new key...\n", keyName)

				// Ask user for key type
				keyType := promptUser("Key type (ed25519/rsa/ecdsa)", "ed25519")

				// Generate key
				var kt models.KeyType
				switch keyType {
				case "ed25519":
					kt = models.KeyTypeED25519
				case "rsa":
					kt = models.KeyTypeRSA
				case "ecdsa":
					kt = models.KeyTypeECDSA
				default:
					return fmt.Errorf("invalid key type: %s", keyType)
				}

				cfg := configManager.Get()
				ks, err := keystore.NewKeyStore(cfg.KeystorePath)
				if err != nil {
					return err
				}

				key, err := ks.GenerateKey(keyName, kt, "", 4096)
				if err != nil {
					return fmt.Errorf("failed to generate key: %w", err)
				}

				if err := configManager.AddKey(*key); err != nil {
					return fmt.Errorf("failed to add key to config: %w", err)
				}

				fmt.Printf("✓ Created new key: %s\n", keyName)
			}
		}

		if keyName == "" {
			return fmt.Errorf("key is required")
		}

		// Verify key exists
		if _, err := configManager.GetKey(keyName); err != nil {
			return errors.New(errors.ErrNotFound, "HOST", fmt.Sprintf("key %s not found", keyName)).
				WithSuggestion(fmt.Sprintf("Create it with: skm key gen --name %s", keyName))
		}

		host := models.Host{
			Host:     hostname,
			User:     user,
			KeyName:  keyName,
			Port:     port,
			Hostname: actualHost,
		}

		if err := configManager.AddHost(host); err != nil {
			return err
		}

		// Update SSH config
		if err := updateSSHConfig(); err != nil {
			return fmt.Errorf("failed to update SSH config: %w", err)
		}

		fmt.Printf("✓ Added host: %s\n", hostname)
		fmt.Printf("  User: %s\n", user)
		fmt.Printf("  Key: %s\n", keyName)
		if port > 0 {
			fmt.Printf("  Port: %d\n", port)
		}
		fmt.Println("\n✓ Updated ~/.ssh/config")

		return nil
	},
}

var hostListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all SSH host configurations",
	RunE: func(cmd *cobra.Command, args []string) error {
		hosts, err := configManager.ListHosts()
		if err != nil {
			return fmt.Errorf("failed to list hosts: %w", err)
		}

		if len(hosts) == 0 {
			fmt.Println("No hosts configured. Add one with: skm host add")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "HOST\tUSER\tKEY\tPORT\tHOSTNAME")
		fmt.Fprintln(w, "----\t----\t---\t----\t--------")

		for _, host := range hosts {
			port := "-"
			if host.Port > 0 {
				port = fmt.Sprintf("%d", host.Port)
			}
			hostname := "-"
			if host.Hostname != "" {
				hostname = host.Hostname
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				host.Host,
				host.User,
				host.KeyName,
				port,
				hostname,
			)
		}

		w.Flush()
		return nil
	},
}

var hostRemoveCmd = &cobra.Command{
	Use:   "remove <hostname>",
	Short: "Remove an SSH host configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]

		if err := configManager.RemoveHost(hostname); err != nil {
			return errors.WrapWithSuggestion(err, errors.ErrNotFound, "HOST",
				fmt.Sprintf("host %s not found", hostname),
				"Run 'skm host list' to see available hosts")
		}

		// Update SSH config
		if err := updateSSHConfig(); err != nil {
			return fmt.Errorf("failed to update SSH config: %w", err)
		}

		fmt.Printf("✓ Removed host: %s\n", hostname)
		fmt.Println("✓ Updated ~/.ssh/config")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(hostCmd)

	// Add command
	hostCmd.AddCommand(hostAddCmd)
	hostAddCmd.Flags().StringP("user", "u", "", "SSH user (required)")
	hostAddCmd.Flags().StringP("key", "k", "", "Key name to use (optional, will auto-create if not specified)")
	hostAddCmd.Flags().IntP("port", "p", 0, "SSH port")
	hostAddCmd.Flags().String("hostname", "", "Actual hostname (if different from host alias)")
	hostAddCmd.Flags().Bool("auto-create-key", false, "Automatically create key if it doesn't exist")
	hostAddCmd.MarkFlagRequired("user")

	// List command
	hostCmd.AddCommand(hostListCmd)

	// Remove command
	hostCmd.AddCommand(hostRemoveCmd)
}
