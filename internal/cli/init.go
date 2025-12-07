package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize SKM configuration",
	Long:  `Initialize SKM by creating the configuration directory and config file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceName, _ := cmd.Flags().GetString("device-name")
		sshDir, _ := cmd.Flags().GetString("ssh-dir")

		if deviceName == "" {
			// Try to get hostname
			hostname, _ := os.Hostname()
			deviceName = hostname
		}

		if err := configManager.Initialize(deviceName, sshDir); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		cfg := configManager.Get()

		fmt.Println("✓ SKM initialized successfully!")
		fmt.Printf("  Device ID: %s\n", cfg.DeviceID)
		fmt.Printf("  Device Name: %s\n", cfg.DeviceName)
		fmt.Printf("  Keystore: %s\n", cfg.KeystorePath)
		fmt.Printf("  SSH Dir: %s\n", cfg.SSHDir)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Generate a new key: skm key gen --name mykey --type ed25519")
		fmt.Println("  2. Add a host: skm host add example.com --user git --key mykey")
		fmt.Println("  3. Bind a Git repo: skm git bind /path/to/repo --host example.com")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("device-name", "n", "", "Device name (defaults to hostname)")
	initCmd.Flags().String("ssh-dir", "", "Custom SSH configuration directory (default ~/.ssh)")
}
