package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/all-dot-files/ssh-key-manager/internal/git"
	"github.com/all-dot-files/ssh-key-manager/internal/keystore"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git integration commands",
	Long:  `Bind Git repositories and manage Git operations with SKM.`,
}

var gitBindCmd = &cobra.Command{
	Use:   "bind <repo-path>",
	Short: "Bind a Git repository to use SKM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := args[0]
		remote, _ := cmd.Flags().GetString("remote")
		host, _ := cmd.Flags().GetString("host")
		user, _ := cmd.Flags().GetString("user")
		keyName, _ := cmd.Flags().GetString("key")
		autoCreate, _ := cmd.Flags().GetBool("auto-create")

		if remote == "" {
			remote = "origin"
		}

		if host == "" {
			return fmt.Errorf("host is required")
		}

		// Resolve absolute path
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Verify it's a git repository
		gitDir := filepath.Join(absPath, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return fmt.Errorf("not a git repository: %s", absPath)
		}

		// Check if host exists, auto-create if enabled
		hostConfig, err := configManager.GetHost(host)
		if err != nil {
			if !autoCreate {
				return fmt.Errorf("host %s not found. Add it with: skm host add %s --user <user> --key <key>\nOr use --auto-create flag", host, host)
			}

			// Auto-create host
			fmt.Printf("Host '%s' not found. Creating new host configuration...\n", host)

			// Determine user
			if user == "" {
				// Try to guess from common hosts
				switch host {
				case "github.com", "gitlab.com", "bitbucket.org":
					user = "git"
				default:
					user = promptUser("SSH user", "")
					if user == "" {
						return fmt.Errorf("user is required")
					}
				}
			}

			// Determine key
			if keyName == "" {
				keyName = fmt.Sprintf("%s-key", host)
			}

			// Check if key exists, create if not
			_, err := configManager.GetKey(keyName)
			if err != nil {
				fmt.Printf("Key '%s' not found. Creating new key...\n", keyName)

				cfg := configManager.Get()
				ks, err := keystore.NewKeyStore(cfg.KeystorePath)
				if err != nil {
					return err
				}

				key, err := ks.GenerateKey(keyName, models.KeyTypeED25519, "", 0)
				if err != nil {
					return fmt.Errorf("failed to generate key: %w", err)
				}

				if err := configManager.AddKey(*key); err != nil {
					return fmt.Errorf("failed to add key to config: %w", err)
				}

				fmt.Printf("✓ Created new ED25519 key: %s\n", keyName)
				fmt.Printf("  Public key: %s\n", key.PubPath)
				fmt.Println("\n⚠️  Don't forget to add the public key to your Git service!")
			}

			// Create host
			newHost := models.Host{
				Host:    host,
				User:    user,
				KeyName: keyName,
			}

			if err := configManager.AddHost(newHost); err != nil {
				return fmt.Errorf("failed to add host: %w", err)
			}

			// Update SSH config
			if err := updateSSHConfig(); err != nil {
				return fmt.Errorf("failed to update SSH config: %w", err)
			}

			fmt.Printf("✓ Created host: %s (user: %s, key: %s)\n", host, user, keyName)

			hostConfig, _ = configManager.GetHost(host)
		}

		// If key is specified, verify it exists
		if keyName != "" {
			if _, err := configManager.GetKey(keyName); err != nil {
				return fmt.Errorf("key %s not found", keyName)
			}
		} else {
			keyName = hostConfig.KeyName
		}

		gitMgr := git.NewManager(configManager)
		if err := gitMgr.BindRepo(absPath, remote, host, user, keyName); err != nil {
			return err
		}

		// Save to config
		repo := models.GitRepo{
			Path:    absPath,
			Remote:  remote,
			Host:    host,
			User:    user,
			KeyName: keyName,
		}

		if err := configManager.AddRepo(repo); err != nil {
			return err
		}

		fmt.Printf("✓ Bound Git repository: %s\n", absPath)
		fmt.Printf("  Remote: %s\n", remote)
		fmt.Printf("  Host: %s\n", host)
		if user != "" {
			fmt.Printf("  User: %s\n", user)
		}
		fmt.Printf("  Key: %s\n", keyName)

		return nil
	},
}

var gitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all bound Git repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := configManager.ListRepos()
		if err != nil {
			return fmt.Errorf("failed to list repos: %w", err)
		}

		if len(repos) == 0 {
			fmt.Println("No Git repositories bound. Bind one with: skm git bind")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PATH\tREMOTE\tHOST\tKEY")
		fmt.Fprintln(w, "----\t------\t----\t---")

		for _, repo := range repos {
			key := repo.KeyName
			if key == "" {
				key = "(from host)"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				repo.Path,
				repo.Remote,
				repo.Host,
				key,
			)
		}

		w.Flush()
		return nil
	},
}

var gitExecCmd = &cobra.Command{
	Use:   "exec <repo-path> -- <git-command>",
	Short: "Execute a Git command with the correct SSH key",
	Long:  `Execute a Git command in a bound repository using the configured SSH key.`,
	Example: `  skm git exec /path/to/repo -- pull
  skm git exec /path/to/repo -- push origin main
  skm git exec . -- fetch --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("usage: skm git exec <repo-path> -- <git-command>")
		}

		repoPath := args[0]
		gitArgs := args[1:]

		// Resolve absolute path
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		gitMgr := git.NewManager(configManager)
		if err := gitMgr.WrapCommand(absPath, gitArgs); err != nil {
			return fmt.Errorf("git command failed: %w", err)
		}

		return nil
	},
}

var gitSSHWrapperCmd = &cobra.Command{
	Use:    "ssh-wrapper",
	Short:  "Internal SSH wrapper for Git (do not call directly)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// This is called by GIT_SSH_COMMAND
		// We need to determine which repo is being accessed and use the right key
		// For now, we'll pass through to regular SSH
		// In a full implementation, we'd parse GIT_DIR or inspect the call stack

		fmt.Fprintln(os.Stderr, "SKM SSH wrapper called - this should be configured via git config")
		return fmt.Errorf("ssh-wrapper not yet fully implemented")
	},
}

var gitHelperCmd = &cobra.Command{
	Use:   "helper",
	Short: "Git credential helper integration",
	Long:  `Manage Git credential helper integration for SSH key management.`,
}

var gitHelperInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install SKM as a Git credential helper",
	Long: `Install SKM as a Git credential helper for automatic SSH key management.

This command configures Git to use SKM as a credential helper, which enables
automatic SSH key selection based on the Git remote host.

Examples:
  # Install helper for specific hosts
  skm git helper install --host github.com --host gitlab.com

  # Install helper globally with host exclusions
  skm git helper install --global --exclude internal.company.com

  # Install helper for a specific repository
  skm git helper install --local

  # Uninstall helper
  skm git helper install --uninstall`,
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		local, _ := cmd.Flags().GetBool("local")
		hosts, _ := cmd.Flags().GetStringSlice("host")
		excludes, _ := cmd.Flags().GetStringSlice("exclude")
		uninstall, _ := cmd.Flags().GetBool("uninstall")
		repoPath, _ := cmd.Flags().GetString("repo")

		if uninstall {
			return uninstallGitHelper(global, local, repoPath)
		}

		if local && global {
			return fmt.Errorf("cannot use both --local and --global")
		}

		// Get SKM binary path
		skmPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get SKM path: %w", err)
		}

		gitMgr := git.NewManager(configManager)

		if local {
			// Install for current repository
			if repoPath == "" {
				repoPath = "."
			}
			absPath, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}

			if err := gitMgr.InstallCredentialHelper(absPath, skmPath, false, hosts, excludes); err != nil {
				return fmt.Errorf("failed to install credential helper: %w", err)
			}
			fmt.Printf("✓ Installed SKM credential helper for repository: %s\n", absPath)
		} else {
			// Install globally
			if err := gitMgr.InstallCredentialHelper("", skmPath, true, hosts, excludes); err != nil {
				return fmt.Errorf("failed to install credential helper: %w", err)
			}
			fmt.Println("✓ Installed SKM credential helper globally")
		}

		if len(hosts) > 0 {
			fmt.Printf("  Active for hosts: %v\n", hosts)
		}
		if len(excludes) > 0 {
			fmt.Printf("  Excluding hosts: %v\n", excludes)
		}

		fmt.Println("\nGit will now automatically use the correct SSH key based on your SKM configuration.")
		fmt.Println("Configure hosts with: skm host add <hostname> --user <user> --key <key>")

		return nil
	},
}

var gitHelperGetCmd = &cobra.Command{
	Use:    "get",
	Short:  "Git credential helper 'get' operation (internal)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gitMgr := git.NewManager(configManager)
		return gitMgr.CredentialHelperGet()
	},
}

var gitHelperStoreCmd = &cobra.Command{
	Use:    "store",
	Short:  "Git credential helper 'store' operation (internal)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gitMgr := git.NewManager(configManager)
		return gitMgr.CredentialHelperStore()
	},
}

var gitHelperEraseCmd = &cobra.Command{
	Use:    "erase",
	Short:  "Git credential helper 'erase' operation (internal)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gitMgr := git.NewManager(configManager)
		return gitMgr.CredentialHelperErase()
	},
}

var gitHelperSSHCommandCmd = &cobra.Command{
	Use:    "ssh-command",
	Short:  "SSH command wrapper for Git helper (internal)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// This is called as core.sshCommand by Git
		// Args will be the SSH arguments passed by Git
		// We need to determine the host and inject the correct key

		gitMgr := git.NewManager(configManager)
		return gitMgr.HandleSSHCommand(args)
	},
}

func uninstallGitHelper(_, local bool, repoPath string) error {
	gitMgr := git.NewManager(configManager)

	if local {
		if repoPath == "" {
			repoPath = "."
		}
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		if err := gitMgr.UninstallCredentialHelper(absPath, false); err != nil {
			return fmt.Errorf("failed to uninstall credential helper: %w", err)
		}
		fmt.Printf("✓ Uninstalled SKM credential helper from repository: %s\n", absPath)
	} else {
		if err := gitMgr.UninstallCredentialHelper("", true); err != nil {
			return fmt.Errorf("failed to uninstall credential helper: %w", err)
		}
		fmt.Println("✓ Uninstalled SKM credential helper globally")
	}

	return nil
}

var gitHookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage global Git hooks",
	Long:  `Install or uninstall global Git hooks to automatically configure SSH keys.`,
}

var gitHookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install global Git hook",
	Long: `Install a global Git hook that automatically configures SSH keys for repositories.
This will set up a pre-push hook that checks if the repository is bound to SKM,
and if not, attempts to auto-configure it based on the remote URL.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configManager.Get()

		// Get SKM binary path
		skmPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get SKM path: %w", err)
		}

		// Create hooks directory
		hooksDir := filepath.Join(cfg.SSHDir, "..", ".git-hooks")
		if err := git.InstallGlobalHook(hooksDir, skmPath); err != nil {
			return fmt.Errorf("failed to install hook: %w", err)
		}

		fmt.Println("✓ Global Git hook installed successfully!")
		fmt.Printf("  Hooks directory: %s\n", hooksDir)
		fmt.Println("\nNow Git operations will automatically use the correct SSH key.")
		fmt.Println("If a repository is not configured, SKM will prompt you to set it up.")

		return nil
	},
}

var gitHookUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall global Git hook",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := git.UninstallGlobalHook(); err != nil {
			return fmt.Errorf("failed to uninstall hook: %w", err)
		}

		fmt.Println("✓ Global Git hook uninstalled successfully!")
		return nil
	},
}

var gitAutoConfigCmd = &cobra.Command{
	Use:    "auto-config <repo-path>",
	Short:  "Auto-configure repository (internal use by hook)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := args[0]

		// Resolve absolute path
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		gitMgr := git.NewManager(configManager)
		if err := gitMgr.AutoConfigureRepo(absPath); err != nil {
			// Not a fatal error, just inform the user
			fmt.Fprintf(os.Stderr, "Note: %v\n", err)
			return nil
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)

	// Bind command
	gitCmd.AddCommand(gitBindCmd)
	gitBindCmd.Flags().StringP("remote", "r", "origin", "Git remote name")
	gitBindCmd.Flags().String("host", "", "SKM host to use (required)")
	gitBindCmd.Flags().StringP("user", "u", "", "Git user (optional, auto-detected for common hosts)")
	gitBindCmd.Flags().StringP("key", "k", "", "Key override (optional, uses host's key by default)")
	gitBindCmd.Flags().Bool("auto-create", false, "Automatically create missing host and key")
	gitBindCmd.MarkFlagRequired("host")

	// List command
	gitCmd.AddCommand(gitListCmd)

	// Exec command
	gitCmd.AddCommand(gitExecCmd)

	// Helper commands
	gitCmd.AddCommand(gitHelperCmd)
	gitHelperInstallCmd.Flags().Bool("global", false, "Install helper globally for all repositories")
	gitHelperInstallCmd.Flags().Bool("local", false, "Install helper for current repository only")
	gitHelperInstallCmd.Flags().StringSlice("host", []string{}, "Only activate helper for specific hosts (can be used multiple times)")
	gitHelperInstallCmd.Flags().StringSlice("exclude", []string{}, "Exclude specific hosts from helper activation (can be used multiple times)")
	gitHelperInstallCmd.Flags().Bool("uninstall", false, "Uninstall the credential helper")
	gitHelperInstallCmd.Flags().String("repo", "", "Repository path for local installation (defaults to current directory)")
	gitHelperCmd.AddCommand(gitHelperInstallCmd)
	gitHelperCmd.AddCommand(gitHelperGetCmd)
	gitHelperCmd.AddCommand(gitHelperStoreCmd)
	gitHelperCmd.AddCommand(gitHelperEraseCmd)
	gitHelperCmd.AddCommand(gitHelperSSHCommandCmd)

	// Hook commands
	gitCmd.AddCommand(gitHookCmd)
	gitHookCmd.AddCommand(gitHookInstallCmd)
	gitHookCmd.AddCommand(gitHookUninstallCmd)

	// Auto-config (hidden)
	gitCmd.AddCommand(gitAutoConfigCmd)

	// SSH wrapper (hidden)
	gitCmd.AddCommand(gitSSHWrapperCmd)
}
