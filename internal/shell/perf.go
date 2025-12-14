package shell

import (
	"time"
)

// Stopwatch tracks elapsed time for completion generation.
type Stopwatch struct {
	start time.Time
}

// NewStopwatch creates and starts a stopwatch.
func NewStopwatch() *Stopwatch {
	return &Stopwatch{start: time.Now()}
}

// Elapsed returns the time since the stopwatch started.
func (s *Stopwatch) Elapsed() time.Duration {
	if s == nil {
		return 0
	}
	return time.Since(s.start)
}

// WithinBudget reports whether elapsed time is within the provided budget.
func (s *Stopwatch) WithinBudget(budget time.Duration) bool {
	return s.Elapsed() <= budget
}
