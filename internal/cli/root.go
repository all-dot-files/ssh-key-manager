package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/all-dot-files/ssh-key-manager/internal/config"
	"github.com/all-dot-files/ssh-key-manager/internal/rotation"
)

var (
	cfgFile       string // Check storage driver consistency (handled in Manager)
	configManager *config.Manager
	debugMode     bool
	verboseMode   bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "skm",
	Short: "SSH Key Manager - Manage your SSH keys with ease",
	Long: `SKM (SSH Key Manager) is a comprehensive tool for managing SSH keys locally,
integrating with Git, and synchronizing keys across devices.

Features:
  - Generate and manage multiple SSH keys (ed25519, rsa, ecdsa)
  - Organize keys with names, tags, and metadata
  - Automatic SSH config management
  - Git repository integration
  - Sync keys across devices via SKM server
  - Secure key storage with encryption`,
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		checkRotation()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if len(os.Args) > 1 {
			if suggestions := rootCmd.SuggestionsFor(os.Args[1]); len(suggestions) > 0 {
				fmt.Fprintf(os.Stderr, "Did you mean:\n")
				for _, s := range suggestions {
					fmt.Fprintf(os.Stderr, "  • %s (try: skm help %s)\n", s, s)
				}
				fmt.Fprintln(os.Stderr)
			}
		}
		fmt.Fprintln(os.Stderr, "Tip: use \"skm help\" or \"skm completion <shell>\" for guidance.")
		PrintError(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.SuggestionsMinimumDistance = 1
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.TraverseChildren = true
	rootCmd.SuggestionsMinimumDistance = 2

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/skm/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug mode with detailed error messages")
	rootCmd.PersistentFlags().BoolVarP(&verboseMode, "verbose", "v", false, "enable verbose output")
}

// initConfig reads in config file
func initConfig() {
	var err error
	configManager, err = config.NewManager(cfgFile)
	if err != nil {
		PrintError(fmt.Errorf("error initializing config: %w", err))
		os.Exit(1)
	}

	// Try to load config (it's okay if it doesn't exist yet)
	_ = configManager.Load()

	// Try to load project config from current directory
	if err := configManager.LoadProjectConfig(""); err != nil && debugMode {
		fmt.Fprintf(os.Stderr, "Debug: Could not load project config: %v\n", err)
	}

	// Override debug mode if set via flag
	if debugMode {
		cfg := configManager.Get()
		cfg.Debug = true
	}

	// Show project config info if verbose
	if verboseMode && configManager.HasProjectConfig() {
		fmt.Fprintf(os.Stderr, "📁 Using project config from: %s\n", configManager.GetProjectPath())
	}
}

// IsDebug returns true if debug mode is enabled
func IsDebug() bool {
	if debugMode {
		return true
	}
	if configManager != nil {
		return configManager.GetEffectiveDebug()
	}
	return false
}

// IsVerbose returns true if verbose mode is enabled
func IsVerbose() bool {
	return verboseMode || debugMode
}

// checkRotation checks if any keys are expired or expiring soon
func checkRotation() {
	if configManager == nil {
		return
	}

	keys, err := configManager.ListKeys()
	if err != nil {
		// Silently ignore errors in post-run check
		return
	}

	if len(keys) == 0 {
		return
	}

	cfg := configManager.Get()
	checker := rotation.NewRotationChecker(cfg.KeyRotationPolicy)

	expiredCount := 0
	warningCount := 0

	for i := range keys {
		if checker.ShouldNotify(&keys[i]) {
			info := checker.CheckKey(&keys[i])
			if info.DaysUntilRotation <= 0 {
				expiredCount++
			} else {
				warningCount++
			}
		}
	}

	if expiredCount > 0 || warningCount > 0 {
		fmt.Fprintf(os.Stderr, "\n⚠️  Warning: %d key(s) expired, %d key(s) expiring soon.\n", expiredCount, warningCount)
		fmt.Fprintln(os.Stderr, "   Run 'skm key rotation-status' for details.")
	}
}
