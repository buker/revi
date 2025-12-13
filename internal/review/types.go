// Package review implements the code review system with multiple specialized modes.
// It provides mode detection (both AI-powered and heuristic), parallel review execution,
// result aggregation, and blocking logic for high-severity issues.
package review

// Mode represents a review mode type
type Mode string

const (
	ModeSecurity    Mode = "security"
	ModePerformance Mode = "performance"
	ModeStyle       Mode = "style"
	ModeErrors      Mode = "errors"
	ModeTesting     Mode = "testing"
	ModeDocs        Mode = "docs"
)

// AllModes returns all available review modes
func AllModes() []Mode {
	return []Mode{
		ModeSecurity,
		ModePerformance,
		ModeStyle,
		ModeErrors,
		ModeTesting,
		ModeDocs,
	}
}

// ModeInfo contains display information for a mode
type ModeInfo struct {
	Name        string
	Description string
}

// GetModeInfo returns display information for a mode
func GetModeInfo(mode Mode) ModeInfo {
	info := map[Mode]ModeInfo{
		ModeSecurity: {
			Name:        "Security",
			Description: "SQL injection, command injection, XSS, auth issues, secrets exposure",
		},
		ModePerformance: {
			Name:        "Performance",
			Description: "N+1 queries, unnecessary loops, allocations, blocking calls, caching",
		},
		ModeStyle: {
			Name:        "Style",
			Description: "Naming conventions, patterns, consistency, idiomatic usage",
		},
		ModeErrors: {
			Name:        "Error Handling",
			Description: "Missing error checks, swallowed exceptions, unhelpful messages",
		},
		ModeTesting: {
			Name:        "Testing",
			Description: "Untested paths, missing assertions, test quality, coverage gaps",
		},
		ModeDocs: {
			Name:        "Documentation",
			Description: "Missing comments, unclear names, outdated comments, API docs",
		},
	}
	return info[mode]
}

// Status represents the status of a review
type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusDone     Status = "done"
	StatusFailed   Status = "failed"
	StatusIssues   Status = "issues_found"
	StatusNoIssues Status = "no_issues"
)

// Issue represents a single issue found during review
type Issue struct {
	Severity    string `json:"severity"` // high, medium, low
	Description string `json:"description"`
	Location    string `json:"location,omitempty"` // file:line if available
}

// Result represents the result of a single review
type Result struct {
	Mode        Mode     `json:"mode"`
	Status      Status   `json:"status"`
	Summary     string   `json:"summary"`
	Issues      []Issue  `json:"issues,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// HasIssues returns true if the result contains issues
func (r *Result) HasIssues() bool {
	return len(r.Issues) > 0
}

// HasHighSeverityIssues returns true if any issues are high severity
func (r *Result) HasHighSeverityIssues() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "high" {
			return true
		}
	}
	return false
}

// DetectionResult represents the result of mode auto-detection
type DetectionResult struct {
	Modes     []Mode `json:"modes"`
	Reasoning string `json:"reasoning"`
}
