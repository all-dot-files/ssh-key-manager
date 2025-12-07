package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage SKM configuration",
	Long:  `View and modify SKM configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configManager.Get()

		fmt.Println("⚙️  SKM Configuration")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Device ID:       %s\n", cfg.DeviceID)
		if cfg.DeviceName != "" {
			fmt.Printf("Device Name:     %s\n", cfg.DeviceName)
		}
		fmt.Printf("Keystore Path:   %s\n", cfg.KeystorePath)
		fmt.Printf("SSH Directory:   %s\n", cfg.SSHDir)
		fmt.Printf("Debug Mode:      %v\n", cfg.Debug)

		if cfg.User != "" {
			fmt.Printf("User:            %s\n", cfg.User)
		}
		if cfg.Email != "" {
			fmt.Printf("Email:           %s\n", cfg.Email)
		}
		if cfg.Server != "" {
			fmt.Printf("Server:          %s\n", cfg.Server)
		}

		fmt.Printf("\nPolicies:\n")
		fmt.Printf("  Default Key Policy:     %s\n", cfg.DefaultKeyPolicy)
		fmt.Printf("  Sync Public Keys:       %v\n", cfg.SyncPolicy.SyncPublicKeys)
		fmt.Printf("  Sync Private Keys:      %v\n", cfg.SyncPolicy.SyncPrivateKeys)
		fmt.Printf("  Require Encryption:     %v\n", cfg.SyncPolicy.RequireEncryption)

		fmt.Printf("\nKey Rotation Policy:\n")
		fmt.Printf("  Enabled:                %v\n", cfg.KeyRotationPolicy.Enabled)
		fmt.Printf("  Max Key Age:            %d months\n", cfg.KeyRotationPolicy.MaxKeyAgeMonths)
		fmt.Printf("  Warn Before:            %d months\n", cfg.KeyRotationPolicy.WarnBeforeMonths)
		fmt.Printf("  Auto Rotate:            %v\n", cfg.KeyRotationPolicy.AutoRotate)
		fmt.Printf("  Notify on Rotation:     %v\n", cfg.KeyRotationPolicy.NotifyOnRotation)

		fmt.Printf("\nData:\n")
		fmt.Printf("  Keys:                   %d\n", len(cfg.Keys))
		fmt.Printf("  Hosts:                  %d\n", len(cfg.Hosts))
		fmt.Printf("  Repos:                  %d\n", len(cfg.Repos))
		fmt.Printf("  Devices:                %d\n", len(cfg.Devices))

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Supports nested keys with dot notation.

Examples:
  skm config set debug true
  skm config set user "John Doe"
  skm config set email john@example.com
  skm config set key_rotation_policy.enabled true
  skm config set key_rotation_policy.max_key_age_months 24`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg := configManager.Get()

		// Handle nested keys
		parts := strings.Split(key, ".")

		switch parts[0] {
		case "debug":
			val, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", value)
			}
			cfg.Debug = val

		case "user":
			cfg.User = value

		case "email":
			cfg.Email = value

		case "device_name":
			cfg.DeviceName = value

		case "server":
			cfg.Server = value

		case "default_key_policy":
			policy := models.KeyPolicy(value)
			if policy != models.KeyPolicyAuto && policy != models.KeyPolicyAsk && policy != models.KeyPolicyNever {
				return fmt.Errorf("invalid key policy: %s (must be auto, ask, or never)", value)
			}
			cfg.DefaultKeyPolicy = policy

		case "key_rotation_policy":
			if len(parts) < 2 {
				return fmt.Errorf("must specify a key rotation policy field")
			}

			switch parts[1] {
			case "enabled":
				val, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				cfg.KeyRotationPolicy.Enabled = val

			case "max_key_age_months":
				val, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid integer value: %s", value)
				}
				cfg.KeyRotationPolicy.MaxKeyAgeMonths = val

			case "warn_before_months":
				val, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid integer value: %s", value)
				}
				cfg.KeyRotationPolicy.WarnBeforeMonths = val

			case "auto_rotate":
				val, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				cfg.KeyRotationPolicy.AutoRotate = val

			case "notify_on_rotation":
				val, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				cfg.KeyRotationPolicy.NotifyOnRotation = val

			default:
				return fmt.Errorf("unknown key rotation policy field: %s", parts[1])
			}

		case "sync_policy":
			if len(parts) < 2 {
				return fmt.Errorf("must specify a sync policy field")
			}

			switch parts[1] {
			case "sync_public_keys":
				val, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				cfg.SyncPolicy.SyncPublicKeys = val

			case "sync_private_keys":
				val, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				cfg.SyncPolicy.SyncPrivateKeys = val

			case "require_encryption":
				val, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				cfg.SyncPolicy.RequireEncryption = val

			default:
				return fmt.Errorf("unknown sync policy field: %s", parts[1])
			}

		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		if err := configManager.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("✓ Set %s = %s\n", key, value)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		cfg := configManager.Get()

		parts := strings.Split(key, ".")

		switch parts[0] {
		case "debug":
			fmt.Println(cfg.Debug)
		case "user":
			fmt.Println(cfg.User)
		case "email":
			fmt.Println(cfg.Email)
		case "device_id":
			fmt.Println(cfg.DeviceID)
		case "device_name":
			fmt.Println(cfg.DeviceName)
		case "server":
			fmt.Println(cfg.Server)
		case "keystore_path":
			fmt.Println(cfg.KeystorePath)
		case "ssh_dir":
			fmt.Println(cfg.SSHDir)
		case "default_key_policy":
			fmt.Println(cfg.DefaultKeyPolicy)

		case "key_rotation_policy":
			if len(parts) < 2 {
				return fmt.Errorf("must specify a key rotation policy field")
			}
			switch parts[1] {
			case "enabled":
				fmt.Println(cfg.KeyRotationPolicy.Enabled)
			case "max_key_age_months":
				fmt.Println(cfg.KeyRotationPolicy.MaxKeyAgeMonths)
			case "warn_before_months":
				fmt.Println(cfg.KeyRotationPolicy.WarnBeforeMonths)
			case "auto_rotate":
				fmt.Println(cfg.KeyRotationPolicy.AutoRotate)
			case "notify_on_rotation":
				fmt.Println(cfg.KeyRotationPolicy.NotifyOnRotation)
			default:
				return fmt.Errorf("unknown key rotation policy field: %s", parts[1])
			}

		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

