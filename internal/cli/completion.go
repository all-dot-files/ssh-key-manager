package cli

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/all-dot-files/ssh-key-manager/internal/shell"
	"github.com/all-dot-files/ssh-key-manager/pkg/logger"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:
  $ source <(skm completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ skm completion bash > /etc/bash_completion.d/skm
  # macOS:
  $ skm completion bash > $(brew --prefix)/etc/bash_completion.d/skm

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ skm completion zsh > "${fpath[1]}/_skm"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ skm completion fish | source

  # To load completions for each session, execute once:
  $ skm completion fish > ~/.config/fish/completions/skm.fish

PowerShell:
  PS> skm completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> skm completion powershell > skm.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		outPath, _ := cmd.Flags().GetString("out")
		clock := shell.NewStopwatch()

		var buf bytes.Buffer
		out := &buf

		switch args[0] {
		case "bash":
			if err := rootCmd.GenBashCompletion(out); err != nil {
				return fmt.Errorf("generate bash completions: %w", err)
			}
		case "zsh":
			if err := rootCmd.GenZshCompletion(out); err != nil {
				return fmt.Errorf("generate zsh completions: %w", err)
			}
		case "fish":
			if err := rootCmd.GenFishCompletion(out, true); err != nil {
				return fmt.Errorf("generate fish completions: %w", err)
			}
		case "powershell":
			if err := rootCmd.GenPowerShellCompletionWithDesc(out); err != nil {
				return fmt.Errorf("generate powershell completions: %w", err)
			}
		}

		if outPath != "" {
			if err := shell.ValidateWritable(outPath); err != nil {
				return fmt.Errorf("cannot write completions: %w", err)
			}
			if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("write completions: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Wrote completions to %s\n", outPath)
		} else {
			if _, err := cmd.OutOrStdout().Write(buf.Bytes()); err != nil {
				return fmt.Errorf("stream completions: %w", err)
			}
		}

		elapsed := clock.Elapsed()
		if !clock.WithinBudget(150 * time.Millisecond) {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠️  completions generated in %s (over 150ms budget)\n", elapsed)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.Flags().String("out", "", "Write completions to the given file path (validates writability)")
}

// Custom completion functions for better autocomplete experience

// ValidKeyNamesFunc returns a function that provides key name completions
func ValidKeyNamesFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	provider := newKeyCompletionProvider(configManager, logger.Log)
	names := provider.Names()
	return names, cobra.ShellCompDirectiveNoFileComp
}

// ValidKeyTypesFunc returns valid key types
func ValidKeyTypesFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"ed25519", "rsa", "ecdsa", "rsa-4096"}, cobra.ShellCompDirectiveNoFileComp
}

// ValidTagsFunc returns existing tags for completion
func ValidTagsFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if configManager == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg := configManager.Get()
	tagSet := make(map[string]bool)
	for _, key := range cfg.Keys {
		for _, tag := range key.Tags {
			tagSet[tag] = true
		}
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags, cobra.ShellCompDirectiveNoFileComp
}

// ValidProjectNamesFunc returns project names for completion
func ValidProjectNamesFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if configManager == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg := configManager.Get()
	return cfg.Projects, cobra.ShellCompDirectiveNoFileComp
}
