package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SyncHistoryEntry represents a single sync operation
type SyncHistoryEntry struct {
	ID              string         `json:"id"`
	Timestamp       time.Time      `json:"timestamp"`
	DeviceID        string         `json:"device_id"`
	Direction       string         `json:"direction"` // "push" or "pull"
	ChangesApplied  int            `json:"changes_applied"`
	ConflictsFound  int            `json:"conflicts_found"`
	Success         bool           `json:"success"`
	Error           string         `json:"error,omitempty"`
	Changes         []KeyChange    `json:"changes"`
	Duration        time.Duration  `json:"duration"`
}

// SyncHistory manages sync history
type SyncHistory struct {
	historyFile string
	entries     []SyncHistoryEntry
	maxEntries  int
}

// NewSyncHistory creates a new sync history manager
func NewSyncHistory(configDir string, maxEntries int) (*SyncHistory, error) {
	historyFile := filepath.Join(configDir, "sync-history.json")

	sh := &SyncHistory{
		historyFile: historyFile,
		maxEntries:  maxEntries,
		entries:     []SyncHistoryEntry{},
	}

	// Load existing history
	_ = sh.Load()

	return sh, nil
}

// Load loads the sync history from disk
func (sh *SyncHistory) Load() error {
	data, err := os.ReadFile(sh.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No history yet
		}
		return fmt.Errorf("failed to read history file: %w", err)
	}

	if err := json.Unmarshal(data, &sh.entries); err != nil {
		return fmt.Errorf("failed to parse history file: %w", err)
	}

	return nil
}

// Save saves the sync history to disk
func (sh *SyncHistory) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(sh.historyFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Sort entries by timestamp (newest first)
	sort.Slice(sh.entries, func(i, j int) bool {
		return sh.entries[i].Timestamp.After(sh.entries[j].Timestamp)
	})

	// Keep only the last N entries
	if len(sh.entries) > sh.maxEntries {
		sh.entries = sh.entries[:sh.maxEntries]
	}

	data, err := json.MarshalIndent(sh.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(sh.historyFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// Add adds a new entry to the history
func (sh *SyncHistory) Add(entry SyncHistoryEntry) error {
	sh.entries = append(sh.entries, entry)
	return sh.Save()
}

// GetRecent returns the N most recent entries
func (sh *SyncHistory) GetRecent(n int) []SyncHistoryEntry {
	if n > len(sh.entries) {
		n = len(sh.entries)
	}

	// Sort by timestamp (newest first)
	sort.Slice(sh.entries, func(i, j int) bool {
		return sh.entries[i].Timestamp.After(sh.entries[j].Timestamp)
	})

	return sh.entries[:n]
}

// GetByID retrieves an entry by ID
func (sh *SyncHistory) GetByID(id string) *SyncHistoryEntry {
	for i := range sh.entries {
		if sh.entries[i].ID == id {
			return &sh.entries[i]
		}
	}
	return nil
}

// GetByDateRange returns entries within a date range
func (sh *SyncHistory) GetByDateRange(start, end time.Time) []SyncHistoryEntry {
	var result []SyncHistoryEntry

	for _, entry := range sh.entries {
		if entry.Timestamp.After(start) && entry.Timestamp.Before(end) {
			result = append(result, entry)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result
}

// GetStats returns statistics about sync history
func (sh *SyncHistory) GetStats() SyncStats {
	stats := SyncStats{
		TotalSyncs: len(sh.entries),
	}

	for _, entry := range sh.entries {
		if entry.Success {
			stats.SuccessfulSyncs++
		} else {
			stats.FailedSyncs++
		}

		stats.TotalChanges += entry.ChangesApplied
		stats.TotalConflicts += entry.ConflictsFound

		if entry.Direction == "push" {
			stats.PushCount++
		} else if entry.Direction == "pull" {
			stats.PullCount++
		}
	}

	if len(sh.entries) > 0 {
		stats.LastSyncTime = sh.entries[0].Timestamp

		for _, entry := range sh.entries {
			if entry.Timestamp.After(stats.LastSyncTime) {
				stats.LastSyncTime = entry.Timestamp
			}
		}
	}

	return stats
}

// SyncStats contains sync statistics
type SyncStats struct {
	TotalSyncs      int       `json:"total_syncs"`
	SuccessfulSyncs int       `json:"successful_syncs"`
	FailedSyncs     int       `json:"failed_syncs"`
	TotalChanges    int       `json:"total_changes"`
	TotalConflicts  int       `json:"total_conflicts"`
	PushCount       int       `json:"push_count"`
	PullCount       int       `json:"pull_count"`
	LastSyncTime    time.Time `json:"last_sync_time"`
}

// FormatHistory formats the history for display
func (sh *SyncHistory) FormatHistory(entries []SyncHistoryEntry) string {
	if len(entries) == 0 {
		return "No sync history found."
	}

	result := "Sync History:\n"
	result += "═════════════════════════════════════════════════════════════\n\n"

	for i, entry := range entries {
		status := "✓"
		if !entry.Success {
			status = "✗"
		}

		result += fmt.Sprintf("%s [%s] %s - %s\n",
			status,
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Direction,
			entry.DeviceID,
		)

		result += fmt.Sprintf("   Changes: %d, Conflicts: %d, Duration: %v\n",
			entry.ChangesApplied,
			entry.ConflictsFound,
			entry.Duration,
		)

		if entry.Error != "" {
			result += fmt.Sprintf("   Error: %s\n", entry.Error)
		}

		if len(entry.Changes) > 0 {
			result += "   Changes:\n"
			for _, change := range entry.Changes {
				var changeSymbol string
				switch change.Type {
				case ChangeTypeCreate:
					changeSymbol = "+"
				case ChangeTypeUpdate:
					changeSymbol = "~"
				case ChangeTypeDelete:
					changeSymbol = "-"
				}
				result += fmt.Sprintf("     %s %s\n", changeSymbol, change.Key.Name)
			}
		}

		if i < len(entries)-1 {
			result += "\n"
		}
	}

	return result
}

// Clear clears all sync history
func (sh *SyncHistory) Clear() error {
	sh.entries = []SyncHistoryEntry{}
	return sh.Save()
}

