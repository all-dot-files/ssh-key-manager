package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/internal/storage"
)

// Store implements storage.Store for SQLite
type Store struct {
	db *sql.DB
}

// NewStore creates a new SQLite store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Host() storage.HostStore {
	return &hostStore{db: s.db}
}

func (s *Store) Key() storage.KeyStore {
	return &keyStore{db: s.db}
}

func (s *Store) Repo() storage.RepoStore {
	return &repoStore{db: s.db}
}

// migrate creates tables if they don't exist
func (s *Store) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS hosts (
			alias TEXT PRIMARY KEY,
			host TEXT NOT NULL,
			user TEXT NOT NULL,
			key_name TEXT NOT NULL,
			port INTEGER,
			hostname TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS keys (
			name TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			path TEXT NOT NULL,
			pub_path TEXT NOT NULL,
			fingerprint TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS repos (
			path TEXT PRIMARY KEY,
			remote TEXT NOT NULL,
			host_alias TEXT NOT NULL,
			user TEXT,
			key_name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	return nil
}

// --- HostStore Implementation ---

type hostStore struct {
	db *sql.DB
}

func (s *hostStore) Add(ctx context.Context, host models.Host) error {
	query := `INSERT INTO hosts (alias, host, user, key_name, port, hostname, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, host.Host, host.Host, host.User, host.KeyName, host.Port, host.Hostname, time.Now())
	return err
}

func (s *hostStore) Get(ctx context.Context, alias string) (*models.Host, error) {
	query := `SELECT alias, host, user, key_name, port, hostname FROM hosts WHERE alias = ?`
	row := s.db.QueryRowContext(ctx, query, alias)

	var h models.Host
	var hostAlias string // we use this to map back to models.Host.Host which is the alias
	err := row.Scan(&hostAlias, &h.Host, &h.User, &h.KeyName, &h.Port, &h.Hostname)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("host not found: %s", alias)
	}
	if err != nil {
		return nil, err
	}
	h.Host = hostAlias // ensure alias is set correctly
	return &h, nil
}

func (s *hostStore) List(ctx context.Context) ([]models.Host, error) {
	query := `SELECT alias, host, user, key_name, port, hostname FROM hosts`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []models.Host
	for rows.Next() {
		var h models.Host
		var hostAlias string
		if err := rows.Scan(&hostAlias, &h.Host, &h.User, &h.KeyName, &h.Port, &h.Hostname); err != nil {
			return nil, err
		}
		h.Host = hostAlias
		hosts = append(hosts, h)
	}
	return hosts, nil
}

func (s *hostStore) Update(ctx context.Context, host models.Host) error {
	query := `UPDATE hosts SET user=?, key_name=?, port=?, hostname=?, updated_at=? WHERE alias=?`
	_, err := s.db.ExecContext(ctx, query, host.User, host.KeyName, host.Port, host.Hostname, time.Now(), host.Host)
	return err
}

func (s *hostStore) Delete(ctx context.Context, alias string) error {
	query := `DELETE FROM hosts WHERE alias = ?`
	_, err := s.db.ExecContext(ctx, query, alias)
	return err
}

// --- KeyStore Implementation ---

type keyStore struct {
	db *sql.DB
}

func (s *keyStore) Add(ctx context.Context, key models.Key) error {
	query := `INSERT INTO keys (name, type, path, pub_path, fingerprint, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, key.Name, key.Type, key.Path, key.PubPath, key.Fingerprint, key.CreatedAt)
	return err
}

func (s *keyStore) Get(ctx context.Context, name string) (*models.Key, error) {
	query := `SELECT name, type, path, pub_path, fingerprint, created_at FROM keys WHERE name = ?`
	row := s.db.QueryRowContext(ctx, query, name)

	var k models.Key
	// models.KeyType is string alias, scan should work
	err := row.Scan(&k.Name, &k.Type, &k.Path, &k.PubPath, &k.Fingerprint, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key not found: %s", name)
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *keyStore) List(ctx context.Context) ([]models.Key, error) {
	query := `SELECT name, type, path, pub_path, fingerprint, created_at FROM keys`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.Key
	for rows.Next() {
		var k models.Key
		if err := rows.Scan(&k.Name, &k.Type, &k.Path, &k.PubPath, &k.Fingerprint, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *keyStore) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM keys WHERE name = ?`
	_, err := s.db.ExecContext(ctx, query, name)
	return err
}

// --- RepoStore Implementation ---

type repoStore struct {
	db *sql.DB
}

func (s *repoStore) Add(ctx context.Context, repo models.GitRepo) error {
	query := `INSERT INTO repos (path, remote, host_alias, user, key_name) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, repo.Path, repo.Remote, repo.Host, repo.User, repo.KeyName)
	return err
}

func (s *repoStore) Get(ctx context.Context, path string) (*models.GitRepo, error) {
	query := `SELECT path, remote, host_alias, user, key_name FROM repos WHERE path = ?`
	row := s.db.QueryRowContext(ctx, query, path)

	var r models.GitRepo
	err := row.Scan(&r.Path, &r.Remote, &r.Host, &r.User, &r.KeyName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repo not found: %s", path)
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *repoStore) List(ctx context.Context) ([]models.GitRepo, error) {
	query := `SELECT path, remote, host_alias, user, key_name FROM repos`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []models.GitRepo
	for rows.Next() {
		var r models.GitRepo
		if err := rows.Scan(&r.Path, &r.Remote, &r.Host, &r.User, &r.KeyName); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, nil
}

func (s *repoStore) Delete(ctx context.Context, path string) error {
	query := `DELETE FROM repos WHERE path = ?`
	_, err := s.db.ExecContext(ctx, query, path)
	return err
}
