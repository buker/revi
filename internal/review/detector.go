// Package review provides code review functionality including mode detection,
// parallel review execution, and result aggregation. It supports multiple
// review modes (security, performance, style, errors, testing, docs) and
// can automatically detect which modes are relevant for a given code diff.
package review

import (
	"context"
	"strings"
)

// Detector defines the interface for detecting which review modes are relevant
// for a given code diff. Implementations analyze the diff content and return
// a list of applicable review modes along with reasoning for the selection.
type Detector interface {
	// Detect analyzes the given diff and returns the relevant review modes,
	// a reasoning string explaining why those modes were selected, and any error.
	// If detection fails, implementations should return a sensible default (e.g., all modes).
	Detect(ctx context.Context, diff string) ([]Mode, string, error)
}

// ClaudeDetector uses Claude AI to intelligently detect relevant review modes
// based on the semantic content of the code diff. It provides more accurate
// detection than heuristic approaches by understanding code context.
type ClaudeDetector struct {
	detectFunc func(ctx context.Context, diff string) (*DetectionResult, error)
}

// NewClaudeDetector creates a new ClaudeDetector with the provided detection function.
// The detectFunc should call Claude to analyze the diff and return detected modes.
func NewClaudeDetector(detectFunc func(ctx context.Context, diff string) (*DetectionResult, error)) *ClaudeDetector {
	return &ClaudeDetector{detectFunc: detectFunc}
}

// Detect analyzes the diff and returns relevant review modes
func (d *ClaudeDetector) Detect(ctx context.Context, diff string) ([]Mode, string, error) {
	result, err := d.detectFunc(ctx, diff)
	if err != nil {
		// On detection failure, fall back to all modes
		return AllModes(), "Auto-detection failed, running all modes", nil
	}

	// Validate and convert modes
	validModes := make([]Mode, 0, len(result.Modes))
	for _, m := range result.Modes {
		if isValidMode(m) {
			validModes = append(validModes, m)
		}
	}

	if len(validModes) == 0 {
		return AllModes(), "No valid modes detected, running all modes", nil
	}

	return validModes, result.Reasoning, nil
}

// isValidMode checks if a mode string is valid
func isValidMode(mode Mode) bool {
	switch mode {
	case ModeSecurity, ModePerformance, ModeStyle, ModeErrors, ModeTesting, ModeDocs:
		return true
	}
	return false
}

// HeuristicDetector uses pattern-matching heuristics to detect relevant review modes.
// It serves as a fallback when AI-based detection is unavailable or fails, scanning
// the diff for keywords associated with security, performance, errors, testing, and docs.
type HeuristicDetector struct{}

// NewHeuristicDetector creates a new HeuristicDetector instance.
func NewHeuristicDetector() *HeuristicDetector {
	return &HeuristicDetector{}
}

// Detect uses heuristics to determine relevant review modes
func (d *HeuristicDetector) Detect(ctx context.Context, diff string) ([]Mode, string, error) {
	diffLower := strings.ToLower(diff)
	var modes []Mode
	var reasons []string

	// Security indicators
	if containsAny(diffLower, []string{
		"password", "secret", "token", "api_key", "apikey",
		"auth", "login", "session", "cookie", "jwt",
		"sql", "query", "exec", "eval", "inject",
		"input", "sanitize", "escape", "validate",
	}) {
		modes = append(modes, ModeSecurity)
		reasons = append(reasons, "security-related code detected")
	}

	// Performance indicators
	if containsAny(diffLower, []string{
		"loop", "for ", "while", "foreach",
		"query", "select", "join", "database", "db.",
		"cache", "memory", "alloc", "buffer",
		"async", "await", "goroutine", "thread",
	}) {
		modes = append(modes, ModePerformance)
		reasons = append(reasons, "performance-sensitive code detected")
	}

	// Error handling indicators
	if containsAny(diffLower, []string{
		"error", "err ", "err.", "exception", "throw",
		"try", "catch", "finally", "panic", "recover",
		"nil", "null", "undefined", "optional",
	}) {
		modes = append(modes, ModeErrors)
		reasons = append(reasons, "error handling code detected")
	}

	// Testing indicators
	if containsAny(diffLower, []string{
		"test", "spec", "assert", "expect", "mock",
		"stub", "fixture", "setup", "teardown",
	}) {
		modes = append(modes, ModeTesting)
		reasons = append(reasons, "test code detected")
	}

	// Documentation indicators
	if containsAny(diffLower, []string{
		"readme", ".md", "comment", "doc", "//",
		"/*", "*/", "\"\"\"", "'''",
	}) {
		modes = append(modes, ModeDocs)
		reasons = append(reasons, "documentation changes detected")
	}

	// Always include style review for non-trivial changes
	if len(diff) > 100 {
		modes = append(modes, ModeStyle)
		reasons = append(reasons, "code style review")
	}

	// If nothing detected, run all modes
	if len(modes) == 0 {
		return AllModes(), "No specific patterns detected, running all modes", nil
	}

	return modes, strings.Join(reasons, ", "), nil
}

// containsAny checks if s contains any of the patterns
func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// FilterModes filters the detected modes based on explicit user-provided flags.
// If any mode is explicitly enabled, only those enabled modes are returned.
// Otherwise, detected modes are filtered by removing any disabled modes.
// This allows users to override automatic detection with explicit mode selection.
func FilterModes(detected []Mode, enabled map[Mode]bool, disabled map[Mode]bool) []Mode {
	// If any mode is explicitly enabled, use only those
	hasExplicitEnabled := false
	for _, v := range enabled {
		if v {
			hasExplicitEnabled = true
			break
		}
	}

	if hasExplicitEnabled {
		var modes []Mode
		for mode, isEnabled := range enabled {
			if isEnabled && !disabled[mode] {
				modes = append(modes, mode)
			}
		}
		return modes
	}

	// Otherwise, filter detected modes by disabled flags
	var modes []Mode
	for _, mode := range detected {
		if !disabled[mode] {
			modes = append(modes, mode)
		}
	}
	return modes
}
