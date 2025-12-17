// Package fix provides functionality for applying suggested code fixes to source files.
// It includes the Applier for applying fixes with path validation, the InteractiveFixer
// for user-driven fix approval, and Stats for tracking fix session results.
package fix

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buker/revi/internal/review"
)

// Applier handles applying fixes to files within a root directory.
type Applier struct {
	root string
}

// NewApplier creates a new Applier that only modifies files within root.
func NewApplier(root string) *Applier {
	return &Applier{root: root}
}

// Apply applies a fix to the file specified in the fix.
// Returns an error if the fix cannot be applied.
func (a *Applier) Apply(fix *review.Fix) error {
	if !fix.Available {
		return fmt.Errorf("fix not available: %s", fix.Reason)
	}

	// Validate the file is within root
	absPath, err := filepath.Abs(fix.FilePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	absRoot, err := filepath.Abs(a.root)
	if err != nil {
		return fmt.Errorf("invalid root path: %w", err)
	}

	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
		return fmt.Errorf("file %s is outside root directory %s", fix.FilePath, a.root)
	}

	// Read the file
	content, err := os.ReadFile(fix.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Get file permissions to preserve them
	info, err := os.Stat(fix.FilePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	perm := info.Mode().Perm()

	// Split into lines
	lines := strings.Split(string(content), "\n")

	// Validate line range
	if fix.StartLine < 1 {
		return fmt.Errorf("start line must be >= 1, got %d", fix.StartLine)
	}
	if fix.EndLine < fix.StartLine {
		return fmt.Errorf("end line (%d) must be >= start line (%d)", fix.EndLine, fix.StartLine)
	}
	// Account for potential trailing newline creating extra empty line
	maxLine := len(lines)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		maxLine = len(lines) - 1
	}
	if fix.EndLine > maxLine {
		return fmt.Errorf("end line (%d) exceeds file length (%d)", fix.EndLine, maxLine)
	}

	// Replace lines (convert to 0-indexed)
	startIdx := fix.StartLine - 1
	endIdx := fix.EndLine - 1

	// Build new content
	var newLines []string
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, fix.Code)
	newLines = append(newLines, lines[endIdx+1:]...)

	// Write back with preserved permissions
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(fix.FilePath, []byte(newContent), perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Preview returns the original and replacement content for the fix.
// The contextLines parameter is reserved for future use to show surrounding
// context; currently it returns only the lines being replaced.
func (a *Applier) Preview(fix *review.Fix, contextLines int) (before, after string, err error) {
	if !fix.Available {
		return "", "", fmt.Errorf("fix not available: %s", fix.Reason)
	}

	file, err := os.Open(fix.FilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open file: %w", err)
	}
	// Close error ignored for read-only file - any significant I/O errors would
	// have been caught during the read operations above
	defer func() { _ = file.Close() }()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	if fix.StartLine < 1 || fix.EndLine > len(lines) {
		return "", "", fmt.Errorf("invalid line range")
	}

	startIdx := fix.StartLine - 1
	endIdx := fix.EndLine - 1

	// Get lines being replaced
	var beforeLines []string
	for i := startIdx; i <= endIdx; i++ {
		beforeLines = append(beforeLines, lines[i])
	}

	before = strings.Join(beforeLines, "\n")
	after = fix.Code

	return before, after, nil
}
