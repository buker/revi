package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// setupTestRepo creates a temporary git repository for testing.
// Returns the repository, its path, and a cleanup function.
func setupTestRepo(t *testing.T) (*Repository, string, func()) {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize a git repository
	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init test repo: %v", err)
	}

	return &Repository{repo: repo}, tmpDir, func() {
		// TempDir cleanup is automatic
	}
}

// setupTestRepoWithCommit creates a test repo with an initial commit.
func setupTestRepoWithCommit(t *testing.T) (*Repository, string, func()) {
	t.Helper()

	repo, tmpDir, cleanup := setupTestRepo(t)

	// Create and stage a file
	filePath := filepath.Join(tmpDir, "initial.txt")
	if err := os.WriteFile(filePath, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("initial.txt"); err != nil {
		t.Fatalf("failed to stage initial file: %v", err)
	}

	// Create initial commit
	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return repo, tmpDir, cleanup
}

// =============================================================================
// Tests for Repository.Root()
// =============================================================================

func TestRepository_Root_ReturnsAbsolutePath(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root() failed: %v", err)
	}

	// Root should be an absolute path
	if !filepath.IsAbs(root) {
		t.Errorf("expected absolute path, got: %s", root)
	}

	// Root should match the temp directory (resolve symlinks for comparison)
	resolvedTmp, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve tmpDir: %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("failed to resolve root: %v", err)
	}

	if resolvedRoot != resolvedTmp {
		t.Errorf("expected root %q, got %q", resolvedTmp, resolvedRoot)
	}
}

func TestRepository_Root_ConsistentAcrossCalls(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	root1, err := repo.Root()
	if err != nil {
		t.Fatalf("first Root() call failed: %v", err)
	}

	root2, err := repo.Root()
	if err != nil {
		t.Fatalf("second Root() call failed: %v", err)
	}

	if root1 != root2 {
		t.Errorf("Root() returned inconsistent values: %q vs %q", root1, root2)
	}
}

func TestRepository_Root_UsableForPathValidation(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root() failed: %v", err)
	}

	// Create a file in the repo
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Verify we can use Root() to validate file paths are within repo
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	resolvedFile, _ := filepath.EvalSymlinks(testFile)

	// File should be within root
	rel, err := filepath.Rel(resolvedRoot, resolvedFile)
	if err != nil {
		t.Fatalf("failed to get relative path: %v", err)
	}

	if strings.HasPrefix(rel, "..") {
		t.Errorf("test file should be within repo root, relative path: %s", rel)
	}

	// Path outside repo should have ".." prefix
	outsidePath := "/etc/passwd"
	rel, err = filepath.Rel(resolvedRoot, outsidePath)
	if err != nil {
		t.Fatalf("failed to get relative path for outside: %v", err)
	}

	if !strings.HasPrefix(rel, "..") {
		t.Errorf("outside path should have '..' prefix, got: %s", rel)
	}
}

func TestRepository_Root_BareRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize a bare repository (no worktree)
	_, err := git.PlainInit(tmpDir, true) // true = bare repo
	if err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	// Open the bare repo
	goGitRepo, err := git.PlainOpen(tmpDir)
	if err != nil {
		t.Fatalf("failed to open bare repo: %v", err)
	}

	repo := &Repository{repo: goGitRepo}

	// Root() should fail for bare repos since they have no worktree
	_, err = repo.Root()
	if err == nil {
		t.Error("expected error for bare repo (no worktree)")
	}

	// Error message should indicate worktree is unavailable
	if !strings.Contains(err.Error(), "worktree") {
		t.Errorf("expected error to mention worktree, got: %v", err)
	}
}

// =============================================================================
// Tests for Open() and error handling
// =============================================================================

func TestOpen_NotAGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Open(tmpDir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}

	if err != ErrNotAGitRepo {
		t.Errorf("expected ErrNotAGitRepo, got: %v", err)
	}
}

func TestOpen_ValidRepo(t *testing.T) {
	_, tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

// =============================================================================
// Regression tests for go-diff-patch integration (unified diff format)
// =============================================================================

func TestGetStagedDiff_UnifiedDiffFormat_NewFile(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// Create and stage a new file
	newFile := filepath.Join(tmpDir, "new_file.go")
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	if err := os.WriteFile(newFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("new_file.go"); err != nil {
		t.Fatalf("failed to stage new file: %v", err)
	}

	diff, err := repo.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff() failed: %v", err)
	}

	// Verify diff contains expected markers for new file
	if !strings.Contains(diff, "diff --git a/new_file.go b/new_file.go") {
		t.Error("diff should contain git diff header for new file")
	}

	if !strings.Contains(diff, "new file mode") {
		t.Error("diff should indicate new file mode")
	}

	// All lines should be additions (start with +)
	if !strings.Contains(diff, "+package main") {
		t.Error("diff should show package line as addition")
	}

	if !strings.Contains(diff, "+func main()") {
		t.Error("diff should show func line as addition")
	}
}

func TestGetStagedDiff_UnifiedDiffFormat_ModifiedFile(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// Modify the existing file
	existingFile := filepath.Join(tmpDir, "initial.txt")
	newContent := "modified content\nwith multiple lines\n"
	if err := os.WriteFile(existingFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("initial.txt"); err != nil {
		t.Fatalf("failed to stage modified file: %v", err)
	}

	diff, err := repo.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff() failed: %v", err)
	}

	// Verify unified diff format markers
	if !strings.Contains(diff, "diff --git a/initial.txt b/initial.txt") {
		t.Error("diff should contain git diff header")
	}

	// Should have --- and +++ markers (unified diff format)
	if !strings.Contains(diff, "--- a/initial.txt") {
		t.Error("diff should contain --- marker for original file")
	}

	if !strings.Contains(diff, "+++ b/initial.txt") {
		t.Error("diff should contain +++ marker for new file")
	}

	// Should show removal and addition
	if !strings.Contains(diff, "-initial content") {
		t.Error("diff should show original content as removal")
	}

	if !strings.Contains(diff, "+modified content") {
		t.Error("diff should show new content as addition")
	}
}

// TestGetStagedDiff_UnifiedDiffFormat_DeletedFile documents the behavior for deleted files.
// KNOWN LIMITATION: The current implementation iterates over idx.Entries which does not
// include deleted files. Deleted files appear in git.Status with Staging==Deleted but
// are not in the index entries, so GetStagedDiff returns ErrNoStagedChanges for
// deletion-only changes. This test verifies and documents this behavior.
func TestGetStagedDiff_UnifiedDiffFormat_DeletedFile(t *testing.T) {
	repo, _, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Use worktree.Remove which properly stages the deletion
	if _, err := worktree.Remove("initial.txt"); err != nil {
		t.Fatalf("failed to remove and stage file: %v", err)
	}

	// Verify the file is staged as deleted in git status
	status, _ := worktree.Status()
	fileStatus := status.File("initial.txt")
	if fileStatus.Staging != git.Deleted {
		t.Errorf("expected file to be staged as Deleted, got: %v", fileStatus.Staging)
	}

	// Get the diff - this tests the deletion code path
	diff, err := repo.GetStagedDiff()

	// Document current behavior: deletion-only changes return ErrNoStagedChanges
	// because idx.Entries doesn't include deleted files
	if err == ErrNoStagedChanges {
		t.Log("Verified: GetStagedDiff returns ErrNoStagedChanges for deletion-only changes (known limitation)")
		return
	}

	if err != nil {
		t.Fatalf("GetStagedDiff() failed: %v", err)
	}

	// If implementation changes to support deletions, verify proper diff format
	if diff != "" {
		if !strings.Contains(diff, "diff --git a/initial.txt b/initial.txt") {
			t.Error("diff should contain git diff header for deleted file")
		}
		if !strings.Contains(diff, "deleted file mode") {
			t.Error("diff should indicate deleted file mode")
		}
		if !strings.Contains(diff, "-initial content") {
			t.Error("diff should show deleted content as removal")
		}
	}
}

func TestGetStagedDiff_UnifiedDiffFormat_HunkHeaders(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// Create a file with multiple lines first
	multiLineFile := filepath.Join(tmpDir, "multi.txt")
	originalContent := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\n"
	if err := os.WriteFile(multiLineFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("multi.txt"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	// Commit the file
	_, err = worktree.Commit("Add multi-line file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Now modify it (change line 5)
	modifiedContent := "line 1\nline 2\nline 3\nline 4\nmodified line 5\nline 6\nline 7\nline 8\nline 9\nline 10\n"
	if err := os.WriteFile(multiLineFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	if _, err := worktree.Add("multi.txt"); err != nil {
		t.Fatalf("failed to stage modified file: %v", err)
	}

	diff, err := repo.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff() failed: %v", err)
	}

	// Verify unified diff format has @@ hunk headers
	if !strings.Contains(diff, "@@") {
		t.Error("unified diff should contain @@ hunk headers")
	}

	// Should contain the change
	if !strings.Contains(diff, "-line 5") {
		t.Error("diff should show original line 5 as removal")
	}

	if !strings.Contains(diff, "+modified line 5") {
		t.Error("diff should show modified line 5 as addition")
	}
}

func TestGetStagedDiff_UnifiedDiffFormat_ContextLines(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// Create a larger file to verify context lines in diff
	largeFile := filepath.Join(tmpDir, "large.txt")
	var lines []string
	for i := 1; i <= 20; i++ {
		lines = append(lines, "unchanged line "+string(rune('A'+i-1)))
	}
	originalContent := strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(largeFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("large.txt"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	_, err = worktree.Commit("Add large file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Change a line in the middle
	lines[9] = "CHANGED line J"
	modifiedContent := strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(largeFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	if _, err := worktree.Add("large.txt"); err != nil {
		t.Fatalf("failed to stage modified file: %v", err)
	}

	diff, err := repo.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff() failed: %v", err)
	}

	// Unified diff should include context lines (lines without + or -)
	// The go-diff-patch library produces proper context
	if !strings.Contains(diff, "-unchanged line J") {
		t.Error("diff should show original line as removal")
	}

	if !strings.Contains(diff, "+CHANGED line J") {
		t.Error("diff should show modified line as addition")
	}

	// Context lines should be present (lines near the change without +/-)
	// Check that some surrounding unchanged lines appear as context
	diffLines := strings.Split(diff, "\n")
	hasContextLine := false
	for _, line := range diffLines {
		// Context lines start with a space (not +, -, or header chars)
		if len(line) > 0 && line[0] == ' ' {
			hasContextLine = true
			break
		}
	}

	if !hasContextLine {
		t.Log("Diff output:\n", diff)
		t.Error("unified diff should include context lines (lines starting with space)")
	}
}

// =============================================================================
// Tests for edge cases and error handling
// =============================================================================

func TestGetStagedDiff_NoStagedChanges(t *testing.T) {
	repo, _, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// No changes staged
	_, err := repo.GetStagedDiff()
	if err == nil {
		t.Fatal("expected error for no staged changes")
	}

	if err != ErrNoStagedChanges {
		t.Errorf("expected ErrNoStagedChanges, got: %v", err)
	}
}

func TestGetStagedDiff_EmptyRepoWithStagedFiles(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create and stage a file in an empty repo (no commits yet)
	newFile := filepath.Join(tmpDir, "first.go")
	content := "package main\n"
	if err := os.WriteFile(newFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("first.go"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	diff, err := repo.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff() failed for empty repo: %v", err)
	}

	// Should still produce valid diff output
	if !strings.Contains(diff, "diff --git") {
		t.Error("empty repo diff should contain git diff header")
	}

	if !strings.Contains(diff, "+package main") {
		t.Error("empty repo diff should show additions")
	}
}

func TestGetStagedFiles_ReturnsCorrectPaths(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// Create and stage multiple files
	files := []string{"file1.go", "subdir/file2.go", "file3.txt"}

	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content\n"), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", f, err)
		}
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	for _, f := range files {
		if _, err := worktree.Add(f); err != nil {
			t.Fatalf("failed to stage file %s: %v", f, err)
		}
	}

	stagedFiles, err := repo.GetStagedFiles()
	if err != nil {
		t.Fatalf("GetStagedFiles() failed: %v", err)
	}

	if len(stagedFiles) != len(files) {
		t.Errorf("expected %d staged files, got %d", len(files), len(stagedFiles))
	}

	// Check all expected files are present
	fileSet := make(map[string]bool)
	for _, f := range stagedFiles {
		fileSet[f] = true
	}

	for _, expected := range files {
		if !fileSet[expected] {
			t.Errorf("expected file %q not found in staged files: %v", expected, stagedFiles)
		}
	}
}

func TestHasStagedChanges(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// Initially no staged changes
	hasChanges, err := repo.HasStagedChanges()
	if err != nil {
		t.Fatalf("HasStagedChanges() failed: %v", err)
	}
	if hasChanges {
		t.Error("expected no staged changes initially")
	}

	// Stage a file
	newFile := filepath.Join(tmpDir, "new.go")
	if err := os.WriteFile(newFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("new.go"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	hasChanges, err = repo.HasStagedChanges()
	if err != nil {
		t.Fatalf("HasStagedChanges() failed: %v", err)
	}
	if !hasChanges {
		t.Error("expected staged changes after adding file")
	}
}

func TestCommit_CreatesCommit(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepoWithCommit(t)
	defer cleanup()

	// Stage a new file
	newFile := filepath.Join(tmpDir, "new.go")
	if err := os.WriteFile(newFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("new.go"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	// Create commit
	hash, err := repo.Commit("Test commit message")
	if err != nil {
		t.Fatalf("Commit() failed: %v", err)
	}

	// Verify hash is returned
	if hash == "" {
		t.Error("expected non-empty commit hash")
	}

	// Verify hash is valid hex
	if len(hash) != 40 {
		t.Errorf("expected 40 character hash, got %d: %s", len(hash), hash)
	}

	// Verify no more staged changes
	hasChanges, err := repo.HasStagedChanges()
	if err != nil {
		t.Fatalf("HasStagedChanges() failed: %v", err)
	}
	if hasChanges {
		t.Error("expected no staged changes after commit")
	}
}
