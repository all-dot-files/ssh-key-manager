package cli

import (
	"log/slog"

	"github.com/all-dot-files/ssh-key-manager/internal/config"
)

// keyCompletionProvider wraps key listing for completions with safe fallbacks.
type keyCompletionProvider struct {
	manager *config.Manager
	log     *slog.Logger
}

func newKeyCompletionProvider(manager *config.Manager, log *slog.Logger) *keyCompletionProvider {
	return &keyCompletionProvider{manager: manager, log: log}
}

func (p *keyCompletionProvider) Names() []string {
	if p.manager == nil {
		return nil
	}
	keys, err := p.manager.ListKeys()
	if err != nil {
		if p.log != nil {
			p.log.Warn("completion key list unavailable", "err", err)
		}
		return nil
	}
	var names []string
	for _, k := range keys {
		if k.Name == "" {
			continue
		}
		names = append(names, k.Name)
	}
	return names
}
