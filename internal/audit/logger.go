package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	EventKeyGenerated    EventType = "key_generated"
	EventKeyDeleted      EventType = "key_deleted"
	EventKeyRotated      EventType = "key_rotated"
	EventKeyExported     EventType = "key_exported"
	EventKeyInstalled    EventType = "key_installed"
	EventDeviceRegistered EventType = "device_registered"
	EventDeviceRevoked   EventType = "device_revoked"
	EventUserLogin       EventType = "user_login"
	EventUserLogout      EventType = "user_logout"
	EventSyncPush        EventType = "sync_push"
	EventSyncPull        EventType = "sync_pull"
	EventConfigChanged   EventType = "config_changed"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType EventType              `json:"event_type"`
	UserID    string                 `json:"user_id,omitempty"`
	DeviceID  string                 `json:"device_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Result    string                 `json:"result"` // success, failure
	Error     string                 `json:"error,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
}

// AuditLogger manages audit logging
type AuditLogger struct {
	logFile    string
	mu         sync.Mutex
	maxEntries int
	entries    []AuditEntry
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(configDir string, maxEntries int) (*AuditLogger, error) {
	logFile := filepath.Join(configDir, "audit.log")

	al := &AuditLogger{
		logFile:    logFile,
		maxEntries: maxEntries,
		entries:    []AuditEntry{},
	}

	// Load existing entries
	_ = al.load()

	return al, nil
}

// Log logs an audit entry
func (al *AuditLogger) Log(entry AuditEntry) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Set timestamp and ID if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.ID == "" {
		entry.ID = generateEntryID()
	}

	al.entries = append(al.entries, entry)

	// Rotate if needed
	if len(al.entries) > al.maxEntries {
		al.entries = al.entries[len(al.entries)-al.maxEntries:]
	}

	return al.save()
}

// LogKeyGenerated logs key generation
func (al *AuditLogger) LogKeyGenerated(deviceID, keyName, keyType string) error {
	return al.Log(AuditEntry{
		EventType: EventKeyGenerated,
		DeviceID:  deviceID,
		Action:    "Generated SSH key",
		Resource:  keyName,
		Details: map[string]interface{}{
			"key_type": keyType,
		},
		Result: "success",
	})
}

// LogKeyDeleted logs key deletion
func (al *AuditLogger) LogKeyDeleted(deviceID, keyName string) error {
	return al.Log(AuditEntry{
		EventType: EventKeyDeleted,
		DeviceID:  deviceID,
		Action:    "Deleted SSH key",
		Resource:  keyName,
		Result:    "success",
	})
}

// LogSync logs synchronization
func (al *AuditLogger) LogSync(deviceID, direction string, changesCount int, success bool) error {
	var eventType EventType
	if direction == "push" {
		eventType = EventSyncPush
	} else {
		eventType = EventSyncPull
	}

	result := "success"
	if !success {
		result = "failure"
	}

	return al.Log(AuditEntry{
		EventType: eventType,
		DeviceID:  deviceID,
		Action:    fmt.Sprintf("Synchronization %s", direction),
		Details: map[string]interface{}{
			"changes_count": changesCount,
		},
		Result: result,
	})
}

// Query queries audit entries
func (al *AuditLogger) Query(filter AuditFilter) []AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	var results []AuditEntry

	for _, entry := range al.entries {
		if filter.Matches(entry) {
			results = append(results, entry)
		}
	}

	return results
}

// GetRecent returns recent entries
func (al *AuditLogger) GetRecent(n int) []AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	if n > len(al.entries) {
		n = len(al.entries)
	}

	// Return last n entries
	return al.entries[len(al.entries)-n:]
}

// GetByDateRange returns entries in date range
func (al *AuditLogger) GetByDateRange(start, end time.Time) []AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	var results []AuditEntry

	for _, entry := range al.entries {
		if entry.Timestamp.After(start) && entry.Timestamp.Before(end) {
			results = append(results, entry)
		}
	}

	return results
}

// GetByDevice returns entries for a device
func (al *AuditLogger) GetByDevice(deviceID string) []AuditEntry {
	return al.Query(AuditFilter{DeviceID: deviceID})
}

// GetByEventType returns entries by event type
func (al *AuditLogger) GetByEventType(eventType EventType) []AuditEntry {
	return al.Query(AuditFilter{EventType: eventType})
}

// Clear clears all audit entries
func (al *AuditLogger) Clear() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.entries = []AuditEntry{}
	return al.save()
}

// load loads audit entries from disk
func (al *AuditLogger) load() error {
	data, err := os.ReadFile(al.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read audit log: %w", err)
	}

	if err := json.Unmarshal(data, &al.entries); err != nil {
		return fmt.Errorf("failed to parse audit log: %w", err)
	}

	return nil
}

// save saves audit entries to disk
func (al *AuditLogger) save() error {
	// Ensure directory exists
	dir := filepath.Dir(al.logFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create audit log directory: %w", err)
	}

	data, err := json.MarshalIndent(al.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit entries: %w", err)
	}

	if err := os.WriteFile(al.logFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// AuditFilter filters audit entries
type AuditFilter struct {
	EventType EventType
	DeviceID  string
	UserID    string
	Resource  string
	StartTime time.Time
	EndTime   time.Time
	Result    string
}

// Matches checks if an entry matches the filter
func (f AuditFilter) Matches(entry AuditEntry) bool {
	if f.EventType != "" && entry.EventType != f.EventType {
		return false
	}

	if f.DeviceID != "" && entry.DeviceID != f.DeviceID {
		return false
	}

	if f.UserID != "" && entry.UserID != f.UserID {
		return false
	}

	if f.Resource != "" && entry.Resource != f.Resource {
		return false
	}

	if !f.StartTime.IsZero() && entry.Timestamp.Before(f.StartTime) {
		return false
	}

	if !f.EndTime.IsZero() && entry.Timestamp.After(f.EndTime) {
		return false
	}

	if f.Result != "" && entry.Result != f.Result {
		return false
	}

	return true
}

// FormatEntries formats audit entries for display
func FormatEntries(entries []AuditEntry) string {
	if len(entries) == 0 {
		return "No audit entries found."
	}

	result := "Audit Log:\n"
	result += "═════════════════════════════════════════════════════════════\n\n"

	for _, entry := range entries {
		status := "✓"
		if entry.Result == "failure" {
			status = "✗"
		}

		result += fmt.Sprintf("%s [%s] %s\n",
			status,
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Action,
		)

		result += fmt.Sprintf("   Event: %s\n", entry.EventType)
		result += fmt.Sprintf("   Device: %s\n", entry.DeviceID)

		if entry.Resource != "" {
			result += fmt.Sprintf("   Resource: %s\n", entry.Resource)
		}

		if len(entry.Details) > 0 {
			result += "   Details:\n"
			for k, v := range entry.Details {
				result += fmt.Sprintf("     %s: %v\n", k, v)
			}
		}

		if entry.Error != "" {
			result += fmt.Sprintf("   Error: %s\n", entry.Error)
		}

		result += "\n"
	}

	return result
}

// generateEntryID generates a unique entry ID
func generateEntryID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}

// GetStatistics returns audit statistics
func (al *AuditLogger) GetStatistics() AuditStatistics {
	al.mu.Lock()
	defer al.mu.Unlock()

	stats := AuditStatistics{
		TotalEntries: len(al.entries),
		EventCounts:  make(map[EventType]int),
	}

	for _, entry := range al.entries {
		stats.EventCounts[entry.EventType]++

		if entry.Result == "success" {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}
	}

	if len(al.entries) > 0 {
		stats.FirstEntry = al.entries[0].Timestamp
		stats.LastEntry = al.entries[len(al.entries)-1].Timestamp
	}

	return stats
}

// AuditStatistics contains audit statistics
type AuditStatistics struct {
	TotalEntries int                  `json:"total_entries"`
	SuccessCount int                  `json:"success_count"`
	FailureCount int                  `json:"failure_count"`
	EventCounts  map[EventType]int    `json:"event_counts"`
	FirstEntry   time.Time            `json:"first_entry"`
	LastEntry    time.Time            `json:"last_entry"`
}

