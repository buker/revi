package review

import (
	"encoding/json"
	"testing"
)

func TestIssueWithFix_JSON(t *testing.T) {
	// Test that Issue can have an optional Fix field that serializes correctly
	issue := Issue{
		Severity:    "high",
		Description: "SQL injection vulnerability",
		Location:    "db/queries.go:42",
		Fix: &Fix{
			Available:   true,
			Code:        `query, args := "SELECT * FROM users WHERE id = $1", []any{userID}`,
			FilePath:    "db/queries.go",
			StartLine:   42,
			EndLine:     42,
			Explanation: "Use parameterized queries to prevent SQL injection",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("failed to marshal issue: %v", err)
	}

	// Unmarshal back
	var decoded Issue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal issue: %v", err)
	}

	// Verify
	if decoded.Fix == nil {
		t.Fatal("expected Fix to be present")
	}
	if !decoded.Fix.Available {
		t.Error("expected Fix.Available to be true")
	}
	if decoded.Fix.FilePath != "db/queries.go" {
		t.Errorf("expected FilePath 'db/queries.go', got %q", decoded.Fix.FilePath)
	}
	if decoded.Fix.StartLine != 42 {
		t.Errorf("expected StartLine 42, got %d", decoded.Fix.StartLine)
	}
}

func TestIssueWithoutFix_JSON(t *testing.T) {
	// Test that Issue without Fix serializes correctly (omits fix field)
	issue := Issue{
		Severity:    "medium",
		Description: "Missing rate limiting",
		Location:    "api/handler.go:88",
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("failed to marshal issue: %v", err)
	}

	// Verify fix is omitted from JSON
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, exists := raw["fix"]; exists {
		t.Error("expected 'fix' field to be omitted when nil")
	}
}

func TestFixUnavailable(t *testing.T) {
	// Test fix with Available=false and reason/alternatives
	fix := Fix{
		Available:    false,
		Reason:       "Requires architectural decision",
		Alternatives: []string{"Add rate limiting middleware", "Use golang.org/x/time/rate package"},
	}

	data, err := json.Marshal(fix)
	if err != nil {
		t.Fatalf("failed to marshal fix: %v", err)
	}

	var decoded Fix
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal fix: %v", err)
	}

	if decoded.Available {
		t.Error("expected Available to be false")
	}
	if decoded.Reason != "Requires architectural decision" {
		t.Errorf("unexpected reason: %q", decoded.Reason)
	}
	if len(decoded.Alternatives) != 2 {
		t.Errorf("expected 2 alternatives, got %d", len(decoded.Alternatives))
	}
}

// =============================================================================
// Tests for Result.HasIssues()
// =============================================================================

func TestResult_HasIssues_True(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusIssues,
		Issues: []Issue{
			{Severity: "high", Description: "issue 1"},
			{Severity: "medium", Description: "issue 2"},
		},
	}

	if !result.HasIssues() {
		t.Error("HasIssues() = false, want true")
	}
}

func TestResult_HasIssues_False(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusNoIssues,
		Issues: []Issue{},
	}

	if result.HasIssues() {
		t.Error("HasIssues() = true, want false")
	}
}

func TestResult_HasIssues_NilIssues(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusNoIssues,
		Issues: nil,
	}

	if result.HasIssues() {
		t.Error("HasIssues() = true, want false for nil issues")
	}
}

// =============================================================================
// Tests for Result.HasHighSeverityIssues()
// =============================================================================

func TestResult_HasHighSeverityIssues_True(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusIssues,
		Issues: []Issue{
			{Severity: "low", Description: "low issue"},
			{Severity: "high", Description: "critical issue"},
			{Severity: "medium", Description: "medium issue"},
		},
	}

	if !result.HasHighSeverityIssues() {
		t.Error("HasHighSeverityIssues() = false, want true")
	}
}

func TestResult_HasHighSeverityIssues_FalseWithMediumLow(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusIssues,
		Issues: []Issue{
			{Severity: "low", Description: "low issue"},
			{Severity: "medium", Description: "medium issue"},
		},
	}

	if result.HasHighSeverityIssues() {
		t.Error("HasHighSeverityIssues() = true, want false (only medium/low)")
	}
}

func TestResult_HasHighSeverityIssues_FalseEmpty(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusNoIssues,
		Issues: []Issue{},
	}

	if result.HasHighSeverityIssues() {
		t.Error("HasHighSeverityIssues() = true, want false (empty issues)")
	}
}

func TestResult_HasHighSeverityIssues_FalseNil(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusNoIssues,
		Issues: nil,
	}

	if result.HasHighSeverityIssues() {
		t.Error("HasHighSeverityIssues() = true, want false (nil issues)")
	}
}

func TestResult_HasHighSeverityIssues_OnlyHighIssue(t *testing.T) {
	result := &Result{
		Mode:   ModeSecurity,
		Status: StatusIssues,
		Issues: []Issue{
			{Severity: "high", Description: "only high"},
		},
	}

	if !result.HasHighSeverityIssues() {
		t.Error("HasHighSeverityIssues() = false, want true")
	}
}

// =============================================================================
// Tests for AllModes()
// =============================================================================

func TestAllModes_ReturnsAllSixModes(t *testing.T) {
	modes := AllModes()

	if len(modes) != 6 {
		t.Errorf("AllModes() returned %d modes, want 6", len(modes))
	}
}

func TestAllModes_ContainsExpectedModes(t *testing.T) {
	modes := AllModes()

	expectedModes := []Mode{
		ModeSecurity,
		ModePerformance,
		ModeStyle,
		ModeErrors,
		ModeTesting,
		ModeDocs,
	}

	modeSet := make(map[Mode]bool)
	for _, m := range modes {
		modeSet[m] = true
	}

	for _, expected := range expectedModes {
		if !modeSet[expected] {
			t.Errorf("AllModes() missing mode %s", expected)
		}
	}
}

func TestAllModes_NoDuplicates(t *testing.T) {
	modes := AllModes()
	seen := make(map[Mode]bool)

	for _, m := range modes {
		if seen[m] {
			t.Errorf("AllModes() contains duplicate mode %s", m)
		}
		seen[m] = true
	}
}

// =============================================================================
// Tests for GetModeInfo()
// =============================================================================

func TestGetModeInfo_AllModesHaveInfo(t *testing.T) {
	for _, mode := range AllModes() {
		t.Run(string(mode), func(t *testing.T) {
			info := GetModeInfo(mode)
			if info.Name == "" {
				t.Errorf("GetModeInfo(%s) has empty Name", mode)
			}
			if info.Description == "" {
				t.Errorf("GetModeInfo(%s) has empty Description", mode)
			}
		})
	}
}
