package rotation

import (
	"fmt"
	"time"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// RotationChecker checks keys for rotation requirements
type RotationChecker struct {
	policy models.KeyRotationPolicy
}

// NewRotationChecker creates a new rotation checker
func NewRotationChecker(policy models.KeyRotationPolicy) *RotationChecker {
	return &RotationChecker{
		policy: policy,
	}
}

// CheckKey checks if a single key needs rotation
func (rc *RotationChecker) CheckKey(key *models.Key) KeyRotationInfo {
	status := key.GetRotationStatus(rc.policy)
	ageMonths := key.GetAgeInMonths()

	info := KeyRotationInfo{
		Key:       key,
		Status:    status,
		AgeMonths: ageMonths,
	}

	// Calculate time until rotation
	baseTime := key.CreatedAt
	if key.LastRotatedAt != nil {
		baseTime = *key.LastRotatedAt
	}

	maxAge := time.Duration(rc.policy.MaxKeyAgeMonths) * 30 * 24 * time.Hour
	rotationDue := baseTime.Add(maxAge)
	info.RotationDue = rotationDue
	info.DaysUntilRotation = int(time.Until(rotationDue).Hours() / 24)

	// Generate message and recommendations
	switch status {
	case models.RotationStatusExpired:
		info.Message = fmt.Sprintf("⚠️  Key '%s' EXPIRED and requires immediate rotation", key.Name)
		info.Recommendations = []string{
			"Rotate this key immediately",
			"Update all services using this key",
			"Consider enabling auto-rotation in your policy",
		}
		info.Priority = PriorityHigh

	case models.RotationStatusWarning:
		info.Message = fmt.Sprintf("⚡ Key '%s' will expire soon", key.Name)
		info.Recommendations = []string{
			fmt.Sprintf("Plan to rotate this key within %d days", info.DaysUntilRotation),
			"Prepare a list of services using this key",
			"Schedule maintenance window for key rotation",
		}
		info.Priority = PriorityMedium

	case models.RotationStatusOK:
		info.Message = fmt.Sprintf("✅ Key '%s' is up to date", key.Name)
		info.Priority = PriorityLow
	}

	return info
}

// CheckAllKeys checks all keys and returns rotation information
func (rc *RotationChecker) CheckAllKeys(keys []models.Key) []KeyRotationInfo {
	results := make([]KeyRotationInfo, 0, len(keys))

	for i := range keys {
		info := rc.CheckKey(&keys[i])
		results = append(results, info)
	}

	return results
}

// GetExpiredKeys returns only keys that require rotation
func (rc *RotationChecker) GetExpiredKeys(keys []models.Key) []KeyRotationInfo {
	var expired []KeyRotationInfo

	for i := range keys {
		info := rc.CheckKey(&keys[i])
		if info.Status == models.RotationStatusExpired {
			expired = append(expired, info)
		}
	}

	return expired
}

// GetWarningKeys returns keys approaching rotation
func (rc *RotationChecker) GetWarningKeys(keys []models.Key) []KeyRotationInfo {
	var warnings []KeyRotationInfo

	for i := range keys {
		info := rc.CheckKey(&keys[i])
		if info.Status == models.RotationStatusWarning {
			warnings = append(warnings, info)
		}
	}

	return warnings
}

// KeyRotationInfo contains rotation information for a key
type KeyRotationInfo struct {
	Key                *models.Key
	Status             models.KeyRotationStatus
	AgeMonths          int
	RotationDue        time.Time
	DaysUntilRotation  int
	Message            string
	Recommendations    []string
	Priority           Priority
}

// Priority represents the urgency of rotation
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityMedium:
		return "Medium"
	case PriorityHigh:
		return "High"
	default:
		return "Unknown"
	}
}

// RotationSummary provides an overview of key rotation status
type RotationSummary struct {
	TotalKeys      int
	ExpiredKeys    int
	WarningKeys    int
	HealthyKeys    int
	OldestKeyAge   int // in months
	NewestKeyAge   int // in months
	AverageKeyAge  float64
}

// GenerateSummary generates a summary of rotation status
func GenerateSummary(infos []KeyRotationInfo) RotationSummary {
	summary := RotationSummary{
		TotalKeys:    len(infos),
		NewestKeyAge: 9999,
	}

	if len(infos) == 0 {
		return summary
	}

	totalAge := 0
	for _, info := range infos {
		switch info.Status {
		case models.RotationStatusExpired:
			summary.ExpiredKeys++
		case models.RotationStatusWarning:
			summary.WarningKeys++
		case models.RotationStatusOK:
			summary.HealthyKeys++
		}

		age := info.AgeMonths
		if age > summary.OldestKeyAge {
			summary.OldestKeyAge = age
		}
		if age < summary.NewestKeyAge {
			summary.NewestKeyAge = age
		}
		totalAge += age
	}

	if summary.TotalKeys > 0 {
		summary.AverageKeyAge = float64(totalAge) / float64(summary.TotalKeys)
	}

	if summary.NewestKeyAge == 9999 {
		summary.NewestKeyAge = 0
	}

	return summary
}

// FormatSummary formats the summary for display
func FormatSummary(summary RotationSummary) string {
	return fmt.Sprintf(`
🔑 Key Rotation Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total Keys:       %d
  ✅ Healthy:        %d
  ⚡ Warning:        %d
  ⚠️  Expired:        %d

📊 Statistics:
  Oldest Key:       %d months
  Newest Key:       %d months
  Average Age:      %.1f months
`,
		summary.TotalKeys,
		summary.HealthyKeys,
		summary.WarningKeys,
		summary.ExpiredKeys,
		summary.OldestKeyAge,
		summary.NewestKeyAge,
		summary.AverageKeyAge,
	)
}

// ShouldNotify determines if user should be notified about rotation
func (rc *RotationChecker) ShouldNotify(key *models.Key) bool {
	if !rc.policy.NotifyOnRotation {
		return false
	}

	status := key.GetRotationStatus(rc.policy)
	return status == models.RotationStatusExpired || status == models.RotationStatusWarning
}

