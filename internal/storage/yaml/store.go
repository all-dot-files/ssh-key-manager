package yaml

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/internal/storage"
	"gopkg.in/yaml.v3"
)

// Store implements storage.Store for YAML file
type Store struct {
	path string
	mu   sync.RWMutex
}

// NewStore creates a new YAML store
func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	// Verify we can read/write or create the file
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Initialize with empty config if not exists
		if err := s.save(&models.Config{
			Version:   "1.0.0",
			UpdatedAt: time.Now(),
		}); err != nil {
			return nil, fmt.Errorf("failed to create initial config: %w", err)
		}
	}
	return s, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) load() (*models.Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	// Initialize slices if nil to avoid nil pointer issues
	if config.Keys == nil {
		config.Keys = []models.Key{}
	}
	if config.Hosts == nil {
		config.Hosts = []models.Host{}
	}
	if config.Repos == nil {
		config.Repos = []models.GitRepo{}
	}

	return &config, nil
}

func (s *Store) save(config *models.Config) error {
	config.UpdatedAt = time.Now()
	
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0600)
}

func (s *Store) Host() storage.HostStore {
	return &hostStore{s}
}

func (s *Store) Key() storage.KeyStore {
	return &keyStore{s}
}

func (s *Store) Repo() storage.RepoStore {
	return &repoStore{s}
}

// --- HostStore ---

type hostStore struct {
	s *Store
}

func (h *hostStore) Add(ctx context.Context, host models.Host) error {
	h.s.mu.Lock()
	defer h.s.mu.Unlock()

	config, err := h.s.load()
	if err != nil {
		return err
	}

	for i, existing := range config.Hosts {
		if existing.Host == host.Host {
			// update existing
			config.Hosts[i] = host
			return h.s.save(config)
		}
	}

	config.Hosts = append(config.Hosts, host)
	return h.s.save(config)
}

func (h *hostStore) Get(ctx context.Context, alias string) (*models.Host, error) {
	h.s.mu.RLock()
	defer h.s.mu.RUnlock()

	config, err := h.s.load()
	if err != nil {
		return nil, err
	}

	for _, host := range config.Hosts {
		if host.Host == alias {
			return &host, nil
		}
	}

	return nil, fmt.Errorf("host not found: %s", alias)
}

func (h *hostStore) List(ctx context.Context) ([]models.Host, error) {
	h.s.mu.RLock()
	defer h.s.mu.RUnlock()

	config, err := h.s.load()
	if err != nil {
		return nil, err
	}
	return config.Hosts, nil
}

func (h *hostStore) Update(ctx context.Context, host models.Host) error {
	return h.Add(ctx, host) // Add handles update/upsert
}

func (h *hostStore) Delete(ctx context.Context, alias string) error {
	h.s.mu.Lock()
	defer h.s.mu.Unlock()

	config, err := h.s.load()
	if err != nil {
		return err
	}

	for i, host := range config.Hosts {
		if host.Host == alias {
			config.Hosts = append(config.Hosts[:i], config.Hosts[i+1:]...)
			return h.s.save(config)
		}
	}

	return fmt.Errorf("host not found: %s", alias)
}

// --- KeyStore ---

type keyStore struct {
	s *Store
}

func (k *keyStore) Add(ctx context.Context, key models.Key) error {
	k.s.mu.Lock()
	defer k.s.mu.Unlock()

	config, err := k.s.load()
	if err != nil {
		return err
	}

	for _, existing := range config.Keys {
		if existing.Name == key.Name {
			return fmt.Errorf("key already exists: %s", key.Name)
		}
	}

	config.Keys = append(config.Keys, key)
	return k.s.save(config)
}

func (k *keyStore) Get(ctx context.Context, name string) (*models.Key, error) {
	k.s.mu.RLock()
	defer k.s.mu.RUnlock()

	config, err := k.s.load()
	if err != nil {
		return nil, err
	}

	for _, key := range config.Keys {
		if key.Name == name {
			return &key, nil
		}
	}

	return nil, fmt.Errorf("key not found: %s", name)
}

func (k *keyStore) List(ctx context.Context) ([]models.Key, error) {
	k.s.mu.RLock()
	defer k.s.mu.RUnlock()

	config, err := k.s.load()
	if err != nil {
		return nil, err
	}
	return config.Keys, nil
}

func (k *keyStore) Delete(ctx context.Context, name string) error {
	k.s.mu.Lock()
	defer k.s.mu.Unlock()

	config, err := k.s.load()
	if err != nil {
		return err
	}

	for i, key := range config.Keys {
		if key.Name == name {
			config.Keys = append(config.Keys[:i], config.Keys[i+1:]...)
			return k.s.save(config)
		}
	}

	return fmt.Errorf("key not found: %s", name)
}

// --- RepoStore ---

type repoStore struct {
	s *Store
}

func (r *repoStore) Add(ctx context.Context, repo models.GitRepo) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	config, err := r.s.load()
	if err != nil {
		return err
	}

	for i, existing := range config.Repos {
		if existing.Path == repo.Path && existing.Remote == repo.Remote {
			config.Repos[i] = repo
			return r.s.save(config)
		}
	}

	config.Repos = append(config.Repos, repo)
	return r.s.save(config)
}

func (r *repoStore) Get(ctx context.Context, path string) (*models.GitRepo, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	config, err := r.s.load()
	if err != nil {
		return nil, err
	}

	for _, repo := range config.Repos {
		if repo.Path == path {
			return &repo, nil
		}
	}

	return nil, fmt.Errorf("repo not found: %s", path)
}

func (r *repoStore) List(ctx context.Context) ([]models.GitRepo, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	config, err := r.s.load()
	if err != nil {
		return nil, err
	}
	return config.Repos, nil
}

func (r *repoStore) Delete(ctx context.Context, path string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	config, err := r.s.load()
	if err != nil {
		return err
	}

	for i, repo := range config.Repos {
		if repo.Path == path {
			config.Repos = append(config.Repos[:i], config.Repos[i+1:]...)
			return r.s.save(config)
		}
	}

	return fmt.Errorf("repo not found: %s", path)
}
