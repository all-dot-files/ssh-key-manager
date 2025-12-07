package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// SyncStrategy defines how to handle sync conflicts
type SyncStrategy string

const (
	StrategyLocalWins  SyncStrategy = "local"
	StrategyRemoteWins SyncStrategy = "remote"
	StrategyNewerWins  SyncStrategy = "newer"
	StrategyManual     SyncStrategy = "manual"
)

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeTypeCreate ChangeType = "create"
	ChangeTypeUpdate ChangeType = "update"
	ChangeTypeDelete ChangeType = "delete"
)

// KeyChange represents a change to a key
type KeyChange struct {
	Type        ChangeType    `json:"type"`
	Key         models.Key    `json:"key"`
	Timestamp   time.Time     `json:"timestamp"`
	DeviceID    string        `json:"device_id"`
	Checksum    string        `json:"checksum"`
}

// SyncState tracks the sync state
type SyncState struct {
	LastSyncTime time.Time            `json:"last_sync_time"`
	DeviceID     string               `json:"device_id"`
	KeyChecksums map[string]string    `json:"key_checksums"` // key name -> checksum
	Changes      []KeyChange          `json:"changes"`
}

// ConflictResolution represents a sync conflict resolution
type ConflictResolution struct {
	KeyName      string       `json:"key_name"`
	LocalKey     models.Key   `json:"local_key"`
	RemoteKey    models.Key   `json:"remote_key"`
	Strategy     SyncStrategy `json:"strategy"`
	ResolvedKey  models.Key   `json:"resolved_key"`
}

// SyncManager manages incremental synchronization
type SyncManager struct {
	deviceID     string
	localState   *SyncState
	remoteState  *SyncState
	strategy     SyncStrategy
}

// NewSyncManager creates a new sync manager
func NewSyncManager(deviceID string, strategy SyncStrategy) *SyncManager {
	return &SyncManager{
		deviceID: deviceID,
		strategy: strategy,
		localState: &SyncState{
			DeviceID:     deviceID,
			KeyChecksums: make(map[string]string),
			Changes:      []KeyChange{},
		},
		remoteState: &SyncState{
			KeyChecksums: make(map[string]string),
			Changes:      []KeyChange{},
		},
	}
}

// ComputeChecksum calculates a checksum for a key
func ComputeChecksum(key models.Key) string {
	// Create a deterministic representation of the key
	data := struct {
		Name        string
		Type        string
		Fingerprint string
		Path        string
		Comment     string
		Tags        []string
		CreatedAt   time.Time
	}{
		Name:        key.Name,
		Type:        string(key.Type),
		Fingerprint: key.Fingerprint,
		Path:        key.Path,
		Comment:     key.Comment,
		Tags:        key.Tags,
		CreatedAt:   key.CreatedAt,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// UpdateLocalState updates the local sync state with current keys
func (sm *SyncManager) UpdateLocalState(keys []models.Key) {
	sm.localState.KeyChecksums = make(map[string]string)

	for _, key := range keys {
		checksum := ComputeChecksum(key)
		sm.localState.KeyChecksums[key.Name] = checksum
	}

	sm.localState.LastSyncTime = time.Now()
}

// UpdateRemoteState updates the remote sync state
func (sm *SyncManager) UpdateRemoteState(state *SyncState) {
	sm.remoteState = state
}

// DetectChanges detects changes between local and remote states
func (sm *SyncManager) DetectChanges(localKeys []models.Key) []KeyChange {
	var changes []KeyChange
	now := time.Now()

	// Create maps for easier lookup
	localMap := make(map[string]models.Key)
	for _, key := range localKeys {
		localMap[key.Name] = key
	}

	// Check for new or updated keys locally
	for name, localChecksum := range sm.localState.KeyChecksums {
		remoteChecksum, existsRemote := sm.remoteState.KeyChecksums[name]

		if !existsRemote {
			// New key locally
			changes = append(changes, KeyChange{
				Type:      ChangeTypeCreate,
				Key:       localMap[name],
				Timestamp: now,
				DeviceID:  sm.deviceID,
				Checksum:  localChecksum,
			})
		} else if localChecksum != remoteChecksum {
			// Updated key
			changes = append(changes, KeyChange{
				Type:      ChangeTypeUpdate,
				Key:       localMap[name],
				Timestamp: now,
				DeviceID:  sm.deviceID,
				Checksum:  localChecksum,
			})
		}
	}

	// Check for deleted keys
	for name := range sm.remoteState.KeyChecksums {
		if _, exists := sm.localState.KeyChecksums[name]; !exists {
			// Key deleted locally
			changes = append(changes, KeyChange{
				Type:      ChangeTypeDelete,
				Key:       models.Key{Name: name},
				Timestamp: now,
				DeviceID:  sm.deviceID,
				Checksum:  "",
			})
		}
	}

	return changes
}

// DetectConflicts detects conflicts between local and remote changes
func (sm *SyncManager) DetectConflicts(localKeys []models.Key, remoteKeys []models.Key) []ConflictResolution {
	var conflicts []ConflictResolution

	localMap := make(map[string]models.Key)
	for _, key := range localKeys {
		localMap[key.Name] = key
	}

	remoteMap := make(map[string]models.Key)
	for _, key := range remoteKeys {
		remoteMap[key.Name] = key
	}

	// Find keys that exist in both but are different
	for name, localKey := range localMap {
		if remoteKey, exists := remoteMap[name]; exists {
			localChecksum := ComputeChecksum(localKey)
			remoteChecksum := ComputeChecksum(remoteKey)

			if localChecksum != remoteChecksum {
				conflicts = append(conflicts, ConflictResolution{
					KeyName:   name,
					LocalKey:  localKey,
					RemoteKey: remoteKey,
					Strategy:  sm.strategy,
				})
			}
		}
	}

	return conflicts
}

// ResolveConflict resolves a conflict based on the strategy
func (sm *SyncManager) ResolveConflict(conflict *ConflictResolution) models.Key {
	switch conflict.Strategy {
	case StrategyLocalWins:
		conflict.ResolvedKey = conflict.LocalKey

	case StrategyRemoteWins:
		conflict.ResolvedKey = conflict.RemoteKey

	case StrategyNewerWins:
		if conflict.LocalKey.UpdatedAt.After(conflict.RemoteKey.UpdatedAt) {
			conflict.ResolvedKey = conflict.LocalKey
		} else {
			conflict.ResolvedKey = conflict.RemoteKey
		}

	case StrategyManual:
		// Manual resolution required - return local key as default
		conflict.ResolvedKey = conflict.LocalKey
	}

	return conflict.ResolvedKey
}

// ApplyChanges applies a list of changes to local keys
func (sm *SyncManager) ApplyChanges(localKeys []models.Key, changes []KeyChange) []models.Key {
	keyMap := make(map[string]models.Key)
	for _, key := range localKeys {
		keyMap[key.Name] = key
	}

	for _, change := range changes {
		switch change.Type {
		case ChangeTypeCreate, ChangeTypeUpdate:
			keyMap[change.Key.Name] = change.Key

		case ChangeTypeDelete:
			delete(keyMap, change.Key.Name)
		}
	}

	// Convert map back to slice
	result := make([]models.Key, 0, len(keyMap))
	for _, key := range keyMap {
		result = append(result, key)
	}

	return result
}

// GetChangelog returns a human-readable changelog
func (sm *SyncManager) GetChangelog(changes []KeyChange) []string {
	var changelog []string

	for _, change := range changes {
		switch change.Type {
		case ChangeTypeCreate:
			changelog = append(changelog, "✨ Created: "+change.Key.Name)
		case ChangeTypeUpdate:
			changelog = append(changelog, "📝 Updated: "+change.Key.Name)
		case ChangeTypeDelete:
			changelog = append(changelog, "🗑️  Deleted: "+change.Key.Name)
		}
	}

	return changelog
}

