package storage

import (
	"context"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// Store aggregates all storage interfaces
type Store interface {
	Host() HostStore
	Key() KeyStore
	Repo() RepoStore
	Close() error
}

// HostStore manages SSH host configurations
type HostStore interface {
	// Add adds a new host
	Add(ctx context.Context, host models.Host) error
	// Get retrieves a host by alias
	Get(ctx context.Context, alias string) (*models.Host, error)
	// List returns all hosts
	List(ctx context.Context) ([]models.Host, error)
	// Update updates an existing host
	Update(ctx context.Context, host models.Host) error
	// Delete deletes a host by alias
	Delete(ctx context.Context, alias string) error
}

// KeyStore manages SSH keys info (metadata, not actual files yet)
// Note: We currently store files on disk, this store tracks them in DB.
type KeyStore interface {
	Add(ctx context.Context, key models.Key) error
	Get(ctx context.Context, name string) (*models.Key, error)
	List(ctx context.Context) ([]models.Key, error)
	Delete(ctx context.Context, name string) error
}

// RepoStore manages Git repository bindings
type RepoStore interface {
	Add(ctx context.Context, repo models.GitRepo) error
	Get(ctx context.Context, path string) (*models.GitRepo, error)
	List(ctx context.Context) ([]models.GitRepo, error)
	Delete(ctx context.Context, path string) error
}
