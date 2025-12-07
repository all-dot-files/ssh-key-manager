package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/all-dot-files/ssh-key-manager/internal/api"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// FileStore implements Store interface using file-based storage
type FileStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStore creates a new file-based store
func NewFileStore(basePath string) (*FileStore, error) {
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	return &FileStore{
		basePath: basePath,
	}, nil
}

// GetUser retrieves a user by username
func (fs *FileStore) GetUser(username string) (*User, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.basePath, "users", username+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (fs *FileStore) GetUserByID(userID string) (*User, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Read all user files to find the one with matching ID
	dir := filepath.Join(fs.basePath, "users")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			continue
		}

		if user.ID == userID {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// CreateUser creates a new user
func (fs *FileStore) CreateUser(user *User) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := filepath.Join(fs.basePath, "users")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := filepath.Join(dir, user.Username+".json")
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// RegisterDevice registers a device for a user
func (fs *FileStore) RegisterDevice(userID string, device *models.Device) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := filepath.Join(fs.basePath, "devices", userID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := filepath.Join(dir, device.ID+".json")
	data, err := json.Marshal(device)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetDevices retrieves all devices for a user
func (fs *FileStore) GetDevices(userID string) ([]models.Device, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	dir := filepath.Join(fs.basePath, "devices", userID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Device{}, nil
		}
		return nil, err
	}

	var devices []models.Device
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var device models.Device
		if err := json.Unmarshal(data, &device); err != nil {
			continue
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// RevokeDevice revokes a device
func (fs *FileStore) RevokeDevice(userID, deviceID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.basePath, "devices", userID, deviceID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var device models.Device
	if err := json.Unmarshal(data, &device); err != nil {
		return err
	}

	device.Revoked = true

	data, err = json.Marshal(device)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// SavePublicKeys saves public keys for a user
func (fs *FileStore) SavePublicKeys(userID string, keys []api.PublicKeyData) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := filepath.Join(fs.basePath, "keys", userID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := filepath.Join(dir, "public_keys.json")
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetPublicKeys retrieves public keys for a user
func (fs *FileStore) GetPublicKeys(userID string) ([]api.PublicKeyData, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.basePath, "keys", userID, "public_keys.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []api.PublicKeyData{}, nil
		}
		return nil, err
	}

	var keys []api.PublicKeyData
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// SavePrivateKeys saves encrypted private keys for a user
func (fs *FileStore) SavePrivateKeys(userID string, keys []api.PrivateKeyData) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := filepath.Join(fs.basePath, "keys", userID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := filepath.Join(dir, "private_keys.json")
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetPrivateKeys retrieves encrypted private keys for a user
func (fs *FileStore) GetPrivateKeys(userID string) ([]api.PrivateKeyData, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.basePath, "keys", userID, "private_keys.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []api.PrivateKeyData{}, nil
		}
		return nil, err
	}

	var keys []api.PrivateKeyData
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// LogAudit logs an audit event
func (fs *FileStore) LogAudit(userID, action, details string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := filepath.Join(fs.basePath, "audit")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	entry := map[string]interface{}{
		"user_id": userID,
		"action":  action,
		"details": details,
		"time":    fmt.Sprintf("%v", os.Stdout), // Timestamp
	}

	data, _ := json.Marshal(entry)

	path := filepath.Join(dir, userID+".log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

// GetAuditLogs retrieves audit logs for a user
func (fs *FileStore) GetAuditLogs(userID string, limit int) ([]interface{}, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.basePath, "audit", userID+".log")
	_, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []interface{}{}, nil
		}
		return nil, err
	}

	// Simple implementation: return empty list
	// In production, parse the log file and return structured data
	return []interface{}{}, nil
}
