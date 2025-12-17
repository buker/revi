package fix

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/buker/revi/internal/review"
)

func TestInteractiveFixer_RunWithFixableIssues(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "high",
			Description: "SQL injection",
			Location:    "db/queries.go:42",
			Fix: &review.Fix{
				Available:   true,
				Code:        `	query := "SELECT * FROM users WHERE id = $1"`,
				FilePath:    "db/queries.go",
				StartLine:   42,
				EndLine:     42,
				Explanation: "Use parameterized queries",
			},
		},
	}
	// User types "y\n" to approve
	input := bytes.NewBufferString("y\n")
	output := &bytes.Buffer{}

	applyCalled := false
	mockApply := func(fix *review.Fix) error {
		applyCalled = true
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	if !applyCalled {
		t.Error("expected Apply to be called")
	}
	if stats.Applied != 1 {
		t.Errorf("expected 1 applied, got %d", stats.Applied)
	}
	if stats.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", stats.Skipped)
	}

	// Check output contains issue info
	outStr := output.String()
	if !strings.Contains(outStr, "SQL injection") {
		t.Error("expected output to contain issue description")
	}
	if !strings.Contains(outStr, "Applied") {
		t.Error("expected output to contain 'Applied'")
	}
}

func TestInteractiveFixer_SkipFix(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "low",
			Description: "Style issue",
			Location:    "main.go:10",
			Fix: &review.Fix{
				Available: true,
				Code:      "newCode",
				FilePath:  "main.go",
				StartLine: 10,
				EndLine:   10,
			},
		},
	}

	// User types "n\n" to skip
	input := bytes.NewBufferString("n\n")
	output := &bytes.Buffer{}

	applyCalled := false
	mockApply := func(fix *review.Fix) error {
		applyCalled = true
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	if applyCalled {
		t.Error("Apply should not be called when user skips")
	}
	if stats.Applied != 0 {
		t.Errorf("expected 0 applied, got %d", stats.Applied)
	}
	if stats.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", stats.Skipped)
	}
}

func TestInteractiveFixer_SkipAll(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "low",
			Description: "Issue 1",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix1",
				FilePath:  "a.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
		{
			Severity:    "low",
			Description: "Issue 2",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix2",
				FilePath:  "b.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
	}

	// User types "s\n" to skip all remaining
	input := bytes.NewBufferString("s\n")
	output := &bytes.Buffer{}

	applyCount := 0
	mockApply := func(fix *review.Fix) error {
		applyCount++
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	if applyCount != 0 {
		t.Errorf("expected 0 applies, got %d", applyCount)
	}
	if stats.Skipped != 2 {
		t.Errorf("expected 2 skipped, got %d", stats.Skipped)
	}
}

func TestInteractiveFixer_UnavailableFix(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "medium",
			Description: "Missing rate limiting",
			Location:    "api/handler.go:88",
			Fix: &review.Fix{
				Available:    false,
				Reason:       "Requires architectural decision",
				Alternatives: []string{"Add middleware", "Use rate limiter package"},
			},
		},
	}

	// User presses Enter to continue
	input := bytes.NewBufferString("\n")
	output := &bytes.Buffer{}

	applyCalled := false
	mockApply := func(fix *review.Fix) error {
		applyCalled = true
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	if applyCalled {
		t.Error("Apply should not be called for unavailable fix")
	}
	if stats.Unfixable != 1 {
		t.Errorf("expected 1 unfixable, got %d", stats.Unfixable)
	}

	// Check output shows reason and alternatives
	outStr := output.String()
	if !strings.Contains(outStr, "Cannot auto-fix") {
		t.Error("expected output to indicate unavailable fix")
	}
	if !strings.Contains(outStr, "Requires architectural decision") {
		t.Error("expected output to contain reason")
	}
	if !strings.Contains(outStr, "Add middleware") {
		t.Error("expected output to contain alternatives")
	}
}

func TestInteractiveFixer_NoFixField(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "low",
			Description: "Some issue without fix",
			Location:    "file.go:1",
			// Fix is nil
		},
	}

	input := bytes.NewBufferString("\n")
	output := &bytes.Buffer{}

	fixer := NewInteractiveFixer(input, output, func(f *review.Fix) error { return nil })
	stats := fixer.Run(issues)

	if stats.Unfixable != 1 {
		t.Errorf("expected 1 unfixable for nil fix, got %d", stats.Unfixable)
	}
}

func TestInteractiveFixer_MixedIssues(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "high",
			Description: "Fixable 1",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix1",
				FilePath:  "a.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
		{
			Severity:    "medium",
			Description: "Unfixable",
			Fix: &review.Fix{
				Available: false,
				Reason:    "Cannot fix",
			},
		},
		{
			Severity:    "low",
			Description: "Fixable 2",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix2",
				FilePath:  "b.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
	}

	// y for first, Enter for unfixable, n for third
	input := bytes.NewBufferString("y\n\nn\n")
	output := &bytes.Buffer{}

	applyCount := 0
	mockApply := func(fix *review.Fix) error {
		applyCount++
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	if applyCount != 1 {
		t.Errorf("expected 1 apply, got %d", applyCount)
	}
	if stats.Applied != 1 {
		t.Errorf("expected 1 applied, got %d", stats.Applied)
	}
	if stats.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", stats.Skipped)
	}
	if stats.Unfixable != 1 {
		t.Errorf("expected 1 unfixable, got %d", stats.Unfixable)
	}
}

func TestInteractiveFixer_EmptyIssues(t *testing.T) {
	input := bytes.NewBufferString("")
	output := &bytes.Buffer{}

	fixer := NewInteractiveFixer(input, output, func(f *review.Fix) error { return nil })
	stats := fixer.Run(nil)

	if stats.Applied != 0 || stats.Skipped != 0 || stats.Unfixable != 0 {
		t.Error("expected all zeros for empty issues")
	}
}

func TestInteractiveFixer_InvalidInput(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "low",
			Description: "Test issue",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix",
				FilePath:  "test.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
	}

	// User types invalid input
	input := bytes.NewBufferString("invalid\n")
	output := &bytes.Buffer{}

	applyCalled := false
	mockApply := func(fix *review.Fix) error {
		applyCalled = true
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	// Invalid input should result in skip
	if applyCalled {
		t.Error("Apply should not be called for invalid input")
	}
	if stats.Skipped != 1 {
		t.Errorf("expected 1 skipped for invalid input, got %d", stats.Skipped)
	}

	// Output should indicate invalid input
	outStr := output.String()
	if !strings.Contains(outStr, "invalid input") {
		t.Error("expected output to indicate invalid input")
	}
}

func TestInteractiveFixer_FullWordInputs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantApplied int
		wantSkipped int
	}{
		{"yes full word", "yes\n", 1, 0},
		{"no full word", "no\n", 0, 1},
		{"skip full word", "skip\n", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := []review.Issue{
				{
					Severity:    "low",
					Description: "Test issue",
					Fix: &review.Fix{
						Available: true,
						Code:      "fix",
						FilePath:  "test.go",
						StartLine: 1,
						EndLine:   1,
					},
				},
			}

			input := bytes.NewBufferString(tt.input)
			output := &bytes.Buffer{}

			mockApply := func(fix *review.Fix) error { return nil }

			fixer := NewInteractiveFixer(input, output, mockApply)
			stats := fixer.Run(issues)

			if stats.Applied != tt.wantApplied {
				t.Errorf("expected %d applied, got %d", tt.wantApplied, stats.Applied)
			}
			if stats.Skipped != tt.wantSkipped {
				t.Errorf("expected %d skipped, got %d", tt.wantSkipped, stats.Skipped)
			}
		})
	}
}

func TestInteractiveFixer_EmptyInputDefaultsToYes(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "low",
			Description: "Test issue",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix",
				FilePath:  "test.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
	}

	// User just presses Enter (empty input after trimming)
	input := bytes.NewBufferString("\n")
	output := &bytes.Buffer{}

	applyCalled := false
	mockApply := func(fix *review.Fix) error {
		applyCalled = true
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	// Empty input should default to "yes" (apply the fix)
	if !applyCalled {
		t.Error("Apply should be called when user presses Enter (empty input)")
	}
	if stats.Applied != 1 {
		t.Errorf("expected 1 applied for empty input, got %d", stats.Applied)
	}
}

func TestInteractiveFixer_ApplyFnReturnsError(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "high",
			Description: "Security vulnerability",
			Location:    "auth/login.go:15",
			Fix: &review.Fix{
				Available:   true,
				Code:        `	return bcrypt.CompareHashAndPassword(hash, password)`,
				FilePath:    "auth/login.go",
				StartLine:   15,
				EndLine:     15,
				Explanation: "Use bcrypt for password comparison",
			},
		},
	}

	// User types "y\n" to approve the fix
	input := bytes.NewBufferString("y\n")
	output := &bytes.Buffer{}

	// Mock applyFn that returns an error (simulating file not found, permission denied, etc.)
	applyError := fmt.Errorf("failed to read file: open auth/login.go: no such file or directory")
	applyCalled := false
	mockApply := func(fix *review.Fix) error {
		applyCalled = true
		return applyError
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	// Verify applyFn was called
	if !applyCalled {
		t.Error("expected Apply to be called")
	}

	// Verify the fix was counted as skipped (not applied) due to error
	if stats.Applied != 0 {
		t.Errorf("expected 0 applied when applyFn errors, got %d", stats.Applied)
	}
	if stats.Skipped != 1 {
		t.Errorf("expected 1 skipped when applyFn errors, got %d", stats.Skipped)
	}

	// Verify output contains failure message
	outStr := output.String()
	if !strings.Contains(outStr, "Failed") {
		t.Error("expected output to contain 'Failed' for apply error")
	}
	if !strings.Contains(outStr, "no such file or directory") {
		t.Error("expected output to contain the error message")
	}
}

func TestInteractiveFixer_EOFDuringPrompt(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "low",
			Description: "Test issue",
			Location:    "test.go:1",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix code",
				FilePath:  "test.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
	}

	// Empty input simulates EOF - reader will return io.EOF when reading
	input := bytes.NewBufferString("")
	output := &bytes.Buffer{}

	applyCalled := false
	mockApply := func(fix *review.Fix) error {
		applyCalled = true
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	// When EOF occurs, prompt() returns "n" (skip), so apply should not be called
	if applyCalled {
		t.Error("Apply should not be called when EOF occurs during prompt")
	}
	// The fix should be counted as skipped due to read error
	if stats.Skipped != 1 {
		t.Errorf("expected 1 skipped when EOF, got %d", stats.Skipped)
	}
	if stats.Applied != 0 {
		t.Errorf("expected 0 applied when EOF, got %d", stats.Applied)
	}
}

func TestInteractiveFixer_ApplyFnErrorInMixedIssues(t *testing.T) {
	issues := []review.Issue{
		{
			Severity:    "high",
			Description: "First issue - will fail",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix1",
				FilePath:  "a.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
		{
			Severity:    "high",
			Description: "Second issue - will succeed",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix2",
				FilePath:  "b.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
		{
			Severity:    "medium",
			Description: "Third issue - will fail",
			Fix: &review.Fix{
				Available: true,
				Code:      "fix3",
				FilePath:  "c.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
	}

	// User approves all three fixes
	input := bytes.NewBufferString("y\ny\ny\n")
	output := &bytes.Buffer{}

	// Mock applyFn that fails for files a.go and c.go, succeeds for b.go
	applyCount := 0
	mockApply := func(fix *review.Fix) error {
		applyCount++
		if fix.FilePath == "a.go" || fix.FilePath == "c.go" {
			return fmt.Errorf("permission denied: %s", fix.FilePath)
		}
		return nil
	}

	fixer := NewInteractiveFixer(input, output, mockApply)
	stats := fixer.Run(issues)

	// Verify all three apply calls were made
	if applyCount != 3 {
		t.Errorf("expected 3 apply calls, got %d", applyCount)
	}

	// Verify stats: 1 applied (b.go), 2 skipped (a.go and c.go failed)
	if stats.Applied != 1 {
		t.Errorf("expected 1 applied, got %d", stats.Applied)
	}
	if stats.Skipped != 2 {
		t.Errorf("expected 2 skipped due to errors, got %d", stats.Skipped)
	}
	if stats.Unfixable != 0 {
		t.Errorf("expected 0 unfixable, got %d", stats.Unfixable)
	}

	// Verify output contains failure messages
	outStr := output.String()
	if !strings.Contains(outStr, "permission denied") {
		t.Error("expected output to contain error message")
	}
}
