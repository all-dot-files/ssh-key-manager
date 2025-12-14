package cli

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/all-dot-files/ssh-key-manager/internal/config"
)

// ConfigManagerForTest sets the global config manager for tests.
func ConfigManagerForTest(mgr *config.Manager) {
	configManager = mgr
}

// CompletionCommandForTest exposes the completion command for testing.
func CompletionCommandForTest() *cobra.Command {
	return completionCmd
}

// NewKeyCompletionProviderForTest exposes the provider for tests.
func NewKeyCompletionProviderForTest(mgr *config.Manager, log *slog.Logger) *keyCompletionProvider {
	return newKeyCompletionProvider(mgr, log)
}

// RootCommandForTest exposes the root command for suggestion checks.
func RootCommandForTest() *cobra.Command {
	return rootCmd
}
