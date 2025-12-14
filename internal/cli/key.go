package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/all-dot-files/ssh-key-manager/internal/keystore"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/internal/rotation"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage SSH keys",
	Long:  `Generate, list, show, and manage SSH keys.`,
}

var keyGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate a new SSH key",
	Long:  `Generate a new SSH key pair and add it to the keystore.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		keyType, _ := cmd.Flags().GetString("type")
		passphrase, _ := cmd.Flags().GetString("passphrase")
		rsaBits, _ := cmd.Flags().GetInt("rsa-bits")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		comment, _ := cmd.Flags().GetString("comment")

		if name == "" {
			return fmt.Errorf("key name is required")
		}

		cfg := configManager.Get()
		ks, err := keystore.NewKeyStore(cfg.KeystorePath)
		if err != nil {
			return err
		}

		// Convert string to KeyType
		var kt models.KeyType
		switch keyType {
		case "ed25519":
			kt = models.KeyTypeED25519
		case "rsa":
			kt = models.KeyTypeRSA
		case "ecdsa":
			kt = models.KeyTypeECDSA
		default:
			return fmt.Errorf("unsupported key type: %s", keyType)
		}

		// Generate key
		key, err := ks.GenerateKey(name, kt, passphrase, rsaBits)
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		key.Tags = tags
		key.Comment = comment

		// Add to config
		if err := configManager.AddKey(*key); err != nil {
			return fmt.Errorf("failed to add key to config: %w", err)
		}

		fmt.Printf("✓ Generated %s key: %s\n", keyType, name)
		fmt.Printf("  Private key: %s\n", key.Path)
		fmt.Printf("  Public key: %s\n", key.PubPath)
		fmt.Printf("  Fingerprint: %s\n", key.Fingerprint)

		return nil
	},
}

var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all SSH keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := configManager.ListKeys()
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}

		if len(keys) == 0 {
			fmt.Println("No keys found. Generate one with: skm key gen")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tFINGERPRINT\tINSTALLED\tCREATED")
		fmt.Fprintln(w, "----\t----\t---\t----\t-------")

		for _, key := range keys {
			installed := "no"
			if key.Installed {
				installed = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				key.Name,
				key.Type,
				key.Fingerprint[:16]+"...",
				installed,
				key.CreatedAt.Format("2006-01-02"),
			)
		}

		w.Flush()
		return nil
	},
}

var keyShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of an SSH key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		key, err := configManager.GetKey(name)
		if err != nil {
			return err
		}

		fmt.Printf("Key: %s\n", key.Name)
		fmt.Printf("Type: %s\n", key.Type)
		fmt.Printf("Fingerprint: %s\n", key.Fingerprint)
		fmt.Printf("Private key: %s\n", key.Path)
		fmt.Printf("Public key: %s\n", key.PubPath)
		fmt.Printf("Installed: %v\n", key.Installed)
		fmt.Printf("Has passphrase: %v\n", key.HasPassphrase)
		if key.Type == models.KeyTypeRSA {
			fmt.Printf("RSA bits: %d\n", key.RSABits)
		}
		if len(key.Tags) > 0 {
			fmt.Printf("Tags: %v\n", key.Tags)
		}
		if key.Comment != "" {
			fmt.Printf("Comment: %s\n", key.Comment)
		}
		fmt.Printf("Created: %s\n", key.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Updated: %s\n", key.UpdatedAt.Format(time.RFC3339))

		// Show public key content
		showPub, _ := cmd.Flags().GetBool("show-public")
		if showPub {
			cfg := configManager.Get()
			ks, _ := keystore.NewKeyStore(cfg.KeystorePath)
			pubKey, err := ks.GetPublicKeyContent(key)
			if err != nil {
				return err
			}
			fmt.Printf("\nPublic Key:\n%s\n", string(pubKey))
		}

		return nil
	},
}

var keyInstallCmd = &cobra.Command{
	Use:   "install <name>",
	Short: "Install a key to ~/.ssh",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		key, err := configManager.GetKey(name)
		if err != nil {
			return err
		}

		cfg := configManager.Get()
		ks, err := keystore.NewKeyStore(cfg.KeystorePath)
		if err != nil {
			return err
		}

		if err := ks.InstallToSSH(key, cfg.SSHDir); err != nil {
			return fmt.Errorf("failed to install key: %w", err)
		}

		fmt.Printf("✓ Installed key %s to %s\n", name, cfg.SSHDir)
		return nil
	},
}

var keyExportCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Export public key to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		output, _ := cmd.Flags().GetString("output")

		if output == "" {
			output = name + ".pub"
		}

		key, err := configManager.GetKey(name)
		if err != nil {
			return err
		}

		cfg := configManager.Get()
		ks, err := keystore.NewKeyStore(cfg.KeystorePath)
		if err != nil {
			return err
		}

		if err := ks.ExportPublicKey(key, output); err != nil {
			return fmt.Errorf("failed to export key: %w", err)
		}

		fmt.Printf("✓ Exported public key to %s\n", output)
		return nil
	},
}

var keyDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an SSH key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Fprintf(cmd.OutOrStdout(), "This will delete key files and remove the config entry for %s.\n", name)
			fmt.Fprint(cmd.OutOrStdout(), "Type the key name to confirm or 'no' to cancel: ")
			var response string
			_, _ = fmt.Fscanln(cmd.InOrStdin(), &response)
			if response != name {
				return fmt.Errorf("delete cancelled; rerun with --force to skip confirmation")
			}
		}

		key, err := configManager.GetKey(name)
		if err != nil {
			return err
		}

		cfg := configManager.Get()
		ks, err := keystore.NewKeyStore(cfg.KeystorePath)
		if err != nil {
			return err
		}

		// Delete from keystore
		if err := ks.DeleteKey(key); err != nil {
			return fmt.Errorf("failed to delete key files: %w", err)
		}

		// Remove from config
		if err := configManager.RemoveKey(name); err != nil {
			return fmt.Errorf("failed to remove key from config: %w", err)
		}

		fmt.Printf("✓ Deleted key: %s\n", name)
		return nil
	},
}

var keyRotationStatusCmd = &cobra.Command{
	Use:   "rotation-status",
	Short: "Check rotation status of all keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := configManager.ListKeys()
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}

		if len(keys) == 0 {
			fmt.Println("No keys found.")
			return nil
		}

		cfg := configManager.Get()
		checker := rotation.NewRotationChecker(cfg.KeyRotationPolicy)
		infos := checker.CheckAllKeys(keys)
		summary := rotation.GenerateSummary(infos)

		fmt.Println(rotation.FormatSummary(summary))

		// Show detailed info for each key
		fmt.Println("📋 Detailed Status:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		for _, info := range infos {
			fmt.Printf("\n%s\n", info.Message)
			fmt.Printf("   Age: %d months", info.AgeMonths)

			if info.Status != models.RotationStatusOK {
				if info.DaysUntilRotation > 0 {
					fmt.Printf(" | Rotation due in: %d days", info.DaysUntilRotation)
				} else {
					fmt.Printf(" | Overdue by: %d days", -info.DaysUntilRotation)
				}
			}
			fmt.Println()

			if len(info.Recommendations) > 0 {
				fmt.Println("   Recommendations:")
				for _, rec := range info.Recommendations {
					fmt.Printf("   • %s\n", rec)
				}
			}
		}

		return nil
	},
}

var keyRotateCmd = &cobra.Command{
	Use:   "rotate <name>",
	Short: "Rotate an SSH key",
	Long:  `Generate a new key to replace an existing one, updating all references.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		keepOld, _ := cmd.Flags().GetBool("keep-old")

		oldKey, err := configManager.GetKey(name)
		if err != nil {
			return err
		}

		cfg := configManager.Get()
		ks, err := keystore.NewKeyStore(cfg.KeystorePath)
		if err != nil {
			return err
		}

		// Generate new key with same parameters
		newName := name + "-rotated-" + time.Now().Format("20060102")
		newKey, err := ks.GenerateKey(newName, oldKey.Type, "", oldKey.RSABits)
		if err != nil {
			return fmt.Errorf("failed to generate new key: %w", err)
		}

		// Copy metadata
		newKey.Tags = oldKey.Tags
		newKey.Comment = oldKey.Comment
		now := time.Now()
		newKey.LastRotatedAt = &now
		newKey.RotatedFrom = oldKey.Name

		// Add new key to config
		if err := configManager.AddKey(*newKey); err != nil {
			return fmt.Errorf("failed to add new key to config: %w", err)
		}

		fmt.Printf("✓ Generated new key: %s\n", newName)
		fmt.Printf("  Fingerprint: %s\n", newKey.Fingerprint)

		if !keepOld {
			fmt.Printf("\n⚠️  Remember to:\n")
			fmt.Printf("  1. Update all services using the old key\n")
			fmt.Printf("  2. Test the new key\n")
			fmt.Printf("  3. Delete the old key with: skm key delete %s\n", name)
		}

		return nil
	},
}

var keyRotateBatchCmd = &cobra.Command{
	Use:   "rotate-batch",
	Short: "Rotate all expired keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := configManager.ListKeys()
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}

		cfg := configManager.Get()
		checker := rotation.NewRotationChecker(cfg.KeyRotationPolicy)
		expired := checker.GetExpiredKeys(keys)

		if len(expired) == 0 {
			fmt.Println("✓ No keys require rotation")
			return nil
		}

		fmt.Printf("Found %d key(s) requiring rotation:\n\n", len(expired))
		for _, info := range expired {
			fmt.Printf("  • %s (age: %d months)\n", info.Key.Name, info.AgeMonths)
		}

		fmt.Print("\nProceed with batch rotation? (y/N): ")
		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}

		ks, err := keystore.NewKeyStore(cfg.KeystorePath)
		if err != nil {
			return err
		}

		successCount := 0
		for _, info := range expired {
			oldKey := info.Key
			newName := oldKey.Name + "-rotated-" + time.Now().Format("20060102")

			newKey, err := ks.GenerateKey(newName, oldKey.Type, "", oldKey.RSABits)
			if err != nil {
				fmt.Printf("✗ Failed to rotate %s: %v\n", oldKey.Name, err)
				continue
			}

			newKey.Tags = oldKey.Tags
			newKey.Comment = oldKey.Comment
			now := time.Now()
			newKey.LastRotatedAt = &now
			newKey.RotatedFrom = oldKey.Name

			if err := configManager.AddKey(*newKey); err != nil {
				fmt.Printf("✗ Failed to save %s: %v\n", newName, err)
				continue
			}

			fmt.Printf("✓ Rotated %s → %s\n", oldKey.Name, newName)
			successCount++
		}

		fmt.Printf("\n✓ Successfully rotated %d/%d keys\n", successCount, len(expired))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(keyCmd)

	// Gen command
	keyCmd.AddCommand(keyGenCmd)
	keyGenCmd.Flags().StringP("name", "n", "", "Key name (required)")
	keyGenCmd.Flags().StringP("type", "t", "ed25519", "Key type (ed25519, rsa, ecdsa)")
	keyGenCmd.Flags().StringP("passphrase", "p", "", "Passphrase to encrypt private key")
	keyGenCmd.Flags().IntP("rsa-bits", "b", 4096, "RSA key size (only for RSA keys)")
	keyGenCmd.Flags().StringSlice("tags", []string{}, "Tags for the key")
	keyGenCmd.Flags().StringP("comment", "c", "", "Comment for the key")
	keyGenCmd.MarkFlagRequired("name")

	// List command
	keyCmd.AddCommand(keyListCmd)

	// Show command
	keyCmd.AddCommand(keyShowCmd)
	keyShowCmd.Flags().Bool("show-public", false, "Show public key content")
	keyShowCmd.ValidArgsFunction = ValidKeyNamesFunc

	// Install command
	keyCmd.AddCommand(keyInstallCmd)
	keyInstallCmd.ValidArgsFunction = ValidKeyNamesFunc

	// Export command
	keyCmd.AddCommand(keyExportCmd)
	keyExportCmd.Flags().StringP("output", "o", "", "Output file path")
	keyExportCmd.ValidArgsFunction = ValidKeyNamesFunc

	// Delete command
	keyCmd.AddCommand(keyDeleteCmd)
	keyDeleteCmd.ValidArgsFunction = ValidKeyNamesFunc
	keyDeleteCmd.Flags().BoolP("force", "f", false, "Delete without interactive confirmation")

	// Rotation commands
	keyCmd.AddCommand(keyRotationStatusCmd)
	keyCmd.AddCommand(keyRotateCmd)
	keyRotateCmd.Flags().Bool("keep-old", false, "Keep the old key after rotation")
	keyRotateCmd.ValidArgsFunction = ValidKeyNamesFunc
	keyCmd.AddCommand(keyRotateBatchCmd)

	// Add completion support for flags
	keyGenCmd.RegisterFlagCompletionFunc("type", ValidKeyTypesFunc)
	keyGenCmd.RegisterFlagCompletionFunc("tags", ValidTagsFunc)
}
