// Package git provides git repository operations using go-git.
// It handles reading staged changes, generating diffs, and creating commits
// without shelling out to the git command-line tool.
package git

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	godiffpatch "github.com/sourcegraph/go-diff-patch"
)

// Sentinel errors for common git operations.
var (
	// ErrNoStagedChanges is returned when attempting to get a diff but no files are staged.
	ErrNoStagedChanges = errors.New("no staged changes found")
	// ErrNotAGitRepo is returned when the path is not a valid git repository.
	ErrNotAGitRepo = errors.New("not a git repository")
)

// Repository wraps a go-git repository and provides high-level operations
// for reading staged changes and creating commits.
type Repository struct {
	repo *git.Repository
}

// Open opens the git repository at the given path.
// Returns ErrNotAGitRepo if the path is not a valid git repository.
func Open(path string) (*Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, ErrNotAGitRepo
		}
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}
	return &Repository{repo: repo}, nil
}

// OpenCurrent opens the git repository in the current working directory.
// This is a convenience wrapper around Open(".").
func OpenCurrent() (*Repository, error) {
	return Open(".")
}

// GetStagedDiff returns a unified diff of all staged changes.
// Returns ErrNoStagedChanges if no files are staged.
// For new repositories without commits, returns the content of staged files as additions.
func (r *Repository) GetStagedDiff() (string, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	// Check if there are staged changes
	hasStagedChanges := false
	for _, s := range status {
		if s.Staging != git.Unmodified && s.Staging != git.Untracked {
			hasStagedChanges = true
			break
		}
	}

	if !hasStagedChanges {
		return "", ErrNoStagedChanges
	}

	// Get HEAD commit tree
	head, err := r.repo.Head()
	if err != nil {
		// No commits yet - all staged files are new
		return r.getStagedFilesContent(status)
	}

	headCommit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get head commit: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get head tree: %w", err)
	}

	// Get the index (staging area)
	idx, err := r.repo.Storer.Index()
	if err != nil {
		return "", fmt.Errorf("failed to get index: %w", err)
	}

	var diffBuilder strings.Builder

	// Compare each staged file with HEAD
	for _, entry := range idx.Entries {
		fileStatus := status.File(entry.Name)
		if fileStatus.Staging == git.Unmodified {
			continue
		}

		diffBuilder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", entry.Name, entry.Name))

		switch fileStatus.Staging {
		case git.Added:
			diffBuilder.WriteString("new file mode 100644\n")
			content, err := r.getIndexFileContent(entry.Hash)
			if err != nil {
				return "", fmt.Errorf("failed to get content for added file %s: %w", entry.Name, err)
			}
			diffBuilder.WriteString(fmt.Sprintf("+++ b/%s\n", entry.Name))
			for _, line := range strings.Split(content, "\n") {
				diffBuilder.WriteString("+" + line + "\n")
			}
		case git.Deleted:
			diffBuilder.WriteString("deleted file mode 100644\n")
			content, err := r.getTreeFileContent(headTree, entry.Name)
			if err != nil {
				return "", fmt.Errorf("failed to get content for deleted file %s: %w", entry.Name, err)
			}
			diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n", entry.Name))
			for _, line := range strings.Split(content, "\n") {
				diffBuilder.WriteString("-" + line + "\n")
			}
		case git.Modified:
			oldContent, err := r.getTreeFileContent(headTree, entry.Name)
			if err != nil {
				return "", fmt.Errorf("failed to get old content for modified file %s: %w", entry.Name, err)
			}
			newContent, err := r.getIndexFileContent(entry.Hash)
			if err != nil {
				return "", fmt.Errorf("failed to get new content for modified file %s: %w", entry.Name, err)
			}
			// Use go-diff-patch library for proper unified diff generation
			patch := godiffpatch.GeneratePatch(entry.Name, oldContent, newContent)
			diffBuilder.WriteString(patch)
		}
		diffBuilder.WriteString("\n")
	}

	return diffBuilder.String(), nil
}

// getStagedFilesContent gets content of all staged files when there's no HEAD
func (r *Repository) getStagedFilesContent(status git.Status) (string, error) {
	idx, err := r.repo.Storer.Index()
	if err != nil {
		return "", err
	}

	var diffBuilder strings.Builder
	for _, entry := range idx.Entries {
		fileStatus := status.File(entry.Name)
		if fileStatus.Staging == git.Unmodified || fileStatus.Staging == git.Untracked {
			continue
		}

		diffBuilder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", entry.Name, entry.Name))
		diffBuilder.WriteString("new file mode 100644\n")

		content, err := r.getIndexFileContent(entry.Hash)
		if err == nil {
			diffBuilder.WriteString(fmt.Sprintf("+++ b/%s\n", entry.Name))
			for _, line := range strings.Split(content, "\n") {
				diffBuilder.WriteString("+" + line + "\n")
			}
		}
		diffBuilder.WriteString("\n")
	}

	return diffBuilder.String(), nil
}

// getIndexFileContent gets file content from the index by hash
func (r *Repository) getIndexFileContent(hash plumbing.Hash) (content string, err error) {
	blob, err := r.repo.BlobObject(hash)
	if err != nil {
		return "", err
	}

	reader, err := blob.Reader()
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// getTreeFileContent gets file content from a tree
func (r *Repository) getTreeFileContent(tree *object.Tree, path string) (string, error) {
	file, err := tree.File(path)
	if err != nil {
		return "", err
	}
	return file.Contents()
}

// GetStagedFiles returns a list of file paths that have staged changes.
// The list includes added, modified, and deleted files.
func (r *Repository) GetStagedFiles() ([]string, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var files []string
	for path, s := range status {
		if s.Staging != git.Unmodified && s.Staging != git.Untracked {
			files = append(files, path)
		}
	}

	return files, nil
}

// Commit creates a new commit with the given message from staged changes.
// Returns the commit hash as a hex string on success.
func (r *Repository) Commit(message string) (string, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	hash, err := worktree.Commit(message, &git.CommitOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	return hash.String(), nil
}

// Root returns the absolute path to the repository root directory.
// This is the top-level directory containing the .git folder, which serves
// as the base for resolving relative file paths within the repository.
func (r *Repository) Root() (string, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root directory (worktree unavailable): %w", err)
	}
	return worktree.Filesystem.Root(), nil
}

// HasStagedChanges returns true if there are any staged changes in the repository.
// This is useful for validating before attempting to create a commit.
func (r *Repository) HasStagedChanges() (bool, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	for _, s := range status {
		if s.Staging != git.Unmodified && s.Staging != git.Untracked {
			return true, nil
		}
	}

	return false, nil
}
