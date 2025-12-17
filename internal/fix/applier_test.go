package fix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buker/revi/internal/review"
)

func TestApplier_Apply_ReplacesLines(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	original := `package main

func main() {
	query := "SELECT * FROM users WHERE id = " + userID
	rows, err := db.Query(query)
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fix := &review.Fix{
		Available: true,
		Code:      `	query, args := "SELECT * FROM users WHERE id = $1", []any{userID}`,
		FilePath:  filePath,
		StartLine: 4,
		EndLine:   4,
	}

	applier := NewApplier(tmpDir)
	if err := applier.Apply(fix); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Read back the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := `package main

func main() {
	query, args := "SELECT * FROM users WHERE id = $1", []any{userID}
	rows, err := db.Query(query)
}
`
	if string(content) != expected {
		t.Errorf("unexpected content:\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}

func TestApplier_Apply_ReplacesMultipleLines(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	original := `package main

func process() {
	// old line 1
	// old line 2
	// old line 3
	return nil
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fix := &review.Fix{
		Available: true,
		Code:      "	// new single line",
		FilePath:  filePath,
		StartLine: 4,
		EndLine:   6,
	}

	applier := NewApplier(tmpDir)
	if err := applier.Apply(fix); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := `package main

func process() {
	// new single line
	return nil
}
`
	if string(content) != expected {
		t.Errorf("unexpected content:\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}

func TestApplier_Apply_RejectsOutsideRoot(t *testing.T) {
	tmpDir := t.TempDir()

	fix := &review.Fix{
		Available: true,
		Code:      "hacked",
		FilePath:  "/etc/passwd",
		StartLine: 1,
		EndLine:   1,
	}

	applier := NewApplier(tmpDir)
	err := applier.Apply(fix)
	if err == nil {
		t.Fatal("expected error for file outside root")
	}
}

func TestApplier_Apply_RejectsInvalidLineRange(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(filePath, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := []struct {
		name      string
		startLine int
		endLine   int
	}{
		{"start zero", 0, 1},
		{"negative start", -1, 1},
		{"end before start", 3, 2},
		{"end past file", 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fix := &review.Fix{
				Available: true,
				Code:      "new",
				FilePath:  filePath,
				StartLine: tt.startLine,
				EndLine:   tt.endLine,
			}

			applier := NewApplier(tmpDir)
			err := applier.Apply(fix)
			if err == nil {
				t.Errorf("expected error for invalid line range")
			}
		})
	}
}

func TestApplier_Apply_PreservesPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "script.sh")

	if err := os.WriteFile(filePath, []byte("#!/bin/bash\necho old\n"), 0755); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fix := &review.Fix{
		Available: true,
		Code:      "echo new",
		FilePath:  filePath,
		StartLine: 2,
		EndLine:   2,
	}

	applier := NewApplier(tmpDir)
	if err := applier.Apply(fix); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Check executable bit is preserved
	if info.Mode().Perm()&0100 == 0 {
		t.Error("expected executable bit to be preserved")
	}
}

func TestApplier_Preview(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	original := `package main

func main() {
	old code here
	return
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fix := &review.Fix{
		Available: true,
		Code:      "new code here",
		FilePath:  filePath,
		StartLine: 4,
		EndLine:   4,
	}

	applier := NewApplier(tmpDir)
	// Note: contextLines parameter is currently reserved for future use.
	// Preview returns only the lines being replaced, not surrounding context.
	before, after, err := applier.Preview(fix, 2)
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}

	// Verify only the target line is returned (no context lines)
	if before != "\told code here" {
		t.Errorf("unexpected before: %q", before)
	}
	if after != "new code here" {
		t.Errorf("unexpected after: %q", after)
	}
}

func TestApplier_Preview_UnavailableFix(t *testing.T) {
	tmpDir := t.TempDir()
	applier := NewApplier(tmpDir)

	fix := &review.Fix{
		Available: false,
		Reason:    "Cannot preview",
	}

	_, _, err := applier.Preview(fix, 2)
	if err == nil {
		t.Error("expected error for unavailable fix")
	}
}

func TestApplier_Preview_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	applier := NewApplier(tmpDir)

	fix := &review.Fix{
		Available: true,
		Code:      "new code",
		FilePath:  filepath.Join(tmpDir, "nonexistent.go"),
		StartLine: 1,
		EndLine:   1,
	}

	_, _, err := applier.Preview(fix, 2)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestApplier_Preview_InvalidLineRange(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(filePath, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := []struct {
		name      string
		startLine int
		endLine   int
	}{
		{"start zero", 0, 1},
		{"end past file", 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fix := &review.Fix{
				Available: true,
				Code:      "new",
				FilePath:  filePath,
				StartLine: tt.startLine,
				EndLine:   tt.endLine,
			}

			applier := NewApplier(tmpDir)
			_, _, err := applier.Preview(fix, 2)
			if err == nil {
				t.Error("expected error for invalid line range")
			}
		})
	}
}

func TestApplier_Apply_UnavailableFix(t *testing.T) {
	tmpDir := t.TempDir()

	fix := &review.Fix{
		Available: false,
		Reason:    "Cannot auto-fix",
	}

	applier := NewApplier(tmpDir)
	err := applier.Apply(fix)
	if err == nil {
		t.Fatal("expected error for unavailable fix")
	}
}

func TestApplier_Apply_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	fix := &review.Fix{
		Available: true,
		Code:      "new code",
		FilePath:  filepath.Join(tmpDir, "nonexistent.go"),
		StartLine: 1,
		EndLine:   1,
	}

	applier := NewApplier(tmpDir)
	err := applier.Apply(fix)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestApplier_Apply_MultiLineReplacementCode(t *testing.T) {
	// Test when fix.Code contains multiple lines (newlines within the replacement code)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	original := `package main

func handler(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	db.Exec("SELECT * FROM users WHERE id = " + data)
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Fix contains multiple lines (multi-line replacement code)
	fix := &review.Fix{
		Available: true,
		Code: `	data := r.URL.Query().Get("data")
	stmt, err := db.Prepare("SELECT * FROM users WHERE id = $1")
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()
	stmt.Exec(data)`,
		FilePath:  filePath,
		StartLine: 4,
		EndLine:   5,
	}

	applier := NewApplier(tmpDir)
	if err := applier.Apply(fix); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := `package main

func handler(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	stmt, err := db.Prepare("SELECT * FROM users WHERE id = $1")
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()
	stmt.Exec(data)
}
`
	if string(content) != expected {
		t.Errorf("unexpected content:\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}
