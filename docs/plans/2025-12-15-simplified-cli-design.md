# Simplified CLI Design

## Summary

Change default behavior so `revi` only generates commit messages. Review functionality moves behind the `revi review` subcommand.

## Command Structure

| Command | Behavior |
|---------|----------|
| `revi` | Generate commit message, confirm, commit |
| `revi commit` | Alias to `revi` |
| `revi review` | Run AI code reviews on staged changes |
| `revi review --fix` | Run reviews + interactively apply fixes |

## Root Command (`revi`)

### Flags Kept
- `-n/--dry-run` - Preview commit message without committing

### Flags Removed
- `-r/--review`, `-R/--no-review`
- `-b/--block`, `-B/--no-block`
- All mode flags (`--security`, `--performance`, etc.)
- `-a/--all`

### Workflow
1. Check for staged changes
2. Get staged diff
3. Generate commit message via Claude
4. Display message and ask for confirmation
5. Create commit (unless dry-run)

## Review Command (`revi review`)

### Flags
- `--fix` / `-f` - Interactively apply fixes
- `-b/--block`, `-B/--no-block` - Control exit code on high-severity issues
- Mode flags: `--security`, `--performance`, `--style`, `--errors`, `--testing`, `--docs`
- `--no-*` variants for disabling specific modes
- `-a/--all` - Run all review modes

### Workflow
1. Analyze staged diff
2. Detect relevant review modes (or use explicit flags)
3. Run reviews in parallel
4. Print results and summary
5. If `--fix`: interactively apply suggested fixes
6. Exit with error if `--block` and high-severity issues found

## Implementation Changes

### `internal/cli/root.go`
- Remove review-related flags from `init()`
- Simplify `runFullWorkflow()` to only: get diff, generate message, confirm, commit
- Remove TUI with review callbacks, use simple text output

### `internal/cli/commit.go`
- Simplify to call root command's logic (alias behavior)
- Remove review execution code

### `internal/cli/review.go`
- Move review-specific flags here (block, mode flags)
- `--fix` already exists, no changes needed

### Files Unchanged
- `internal/claude/` - Client stays the same
- `internal/review/` - Review logic stays the same
- `internal/fix/` - Fix logic stays the same
- `internal/git/` - Git operations stay the same
