# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**revi** is an AI-powered code review and commit message generator CLI tool that uses the Claude CLI to analyze staged git changes. It runs specialized code reviews (security, performance, style, errors, testing, docs) in parallel and generates conventional commit messages.

## Build Commands

**Always use the Makefile for build, test, and lint commands.** Do not run `go build`, `go test`, or other Go commands directly - use the corresponding `make` targets instead.

```bash
make build          # Build optimized binary to bin/revi
make dev            # Build development version (no optimization)
make test           # Run all tests
make lint           # Run golangci-lint (must be installed)
make install        # Install to ~/.local/bin
make clean          # Remove build artifacts
```

Run a single test:
```bash
go test -v ./internal/review -run TestFunctionName
```

## Architecture

### Data Flow
1. `cmd/revi/main.go` → `cli.Execute()` - Entry point
2. `internal/cli/root.go` - Cobra command handling, orchestrates full workflow
3. `internal/git/` - Uses go-git library (not shell commands) for repo operations
4. `internal/claude/client.go` - Invokes Claude CLI with JSON prompts, parses responses
5. `internal/review/` - Mode detection (AI + heuristic fallback), parallel execution via Runner
6. `internal/tui/` - Bubble Tea UI for real-time progress display
7. `internal/config/` - Viper-based config from `.revi.yaml`, env vars (`REVI_*`), flags

### Key Patterns

**Claude CLI Integration** (`internal/claude/client.go`):
- Uses `claude -p --output-format json` with prompts via stdin
- Parses JSON wrapper response, extracts inner JSON from potential markdown code blocks
- Handles nested braces and string escaping in JSON extraction

**Review Modes** (`internal/review/types.go`):
- Six modes: `security`, `performance`, `style`, `errors`, `testing`, `docs`
- Each review returns structured `Result` with `Issue` items containing optional `Fix` suggestions
- High-severity issues block commits by default (configurable)

**Parallel Review Execution** (`internal/review/runner.go`):
- `Runner.Run()` executes all modes concurrently with goroutines
- Status callbacks update TUI in real-time
- Results collected via sync.WaitGroup

**TUI Communication** (`internal/tui/program.go`):
- Bubble Tea program runs in background goroutine
- Main workflow sends messages via `Program.Send()` for state updates
- `RunWithCallbacks()` orchestrates detection → reviews → commit generation

### Configuration Priority
1. Command-line flags (highest)
2. Environment variables (`REVI_CLAUDE_PATH`, `REVI_REVIEW_BLOCK`, etc.)
3. `.revi.yaml` in current directory
4. `~/.revi.yaml` global config (lowest)

## Dependencies

- **cobra/viper** - CLI framework and configuration
- **go-git** - Pure Go git operations (no shell)
- **go-diff-patch** - Git-compatible unified diff generation (uses Myers algorithm)
- **bubbletea/bubbles/lipgloss** - Terminal UI
- **Claude CLI** - External dependency, must be installed separately

## Development Guidelines

### Do Not Modify go.mod Version

**Never change the Go version in `go.mod` without explicit user request.** The project requires a specific Go version for compatibility and testing. If build issues occur due to Go version constraints, report the issue to the user rather than modifying the version.

### Prefer Existing Libraries Over Custom Implementations

**Always research and use existing, well-tested libraries before implementing custom solutions.** This applies especially to:

- **Diff/patch generation**: Use `github.com/sourcegraph/go-diff-patch` for unified diffs, not custom line comparison
- **Git operations**: Use `go-git` methods, not shell commands or manual implementations
- **Text processing**: Check if `strings`, `bytes`, or specialized libraries (e.g., `text/template`) already solve the problem

**Why this matters:**
- Existing libraries are battle-tested with edge cases handled
- Custom implementations often have subtle bugs (e.g., naive diff algorithms miss context, create incorrect patches)
- Maintenance burden shifts to library maintainers
- Better performance through optimized algorithms (e.g., Myers diff vs line-by-line comparison)

**Example:** This project uses `github.com/sourcegraph/go-diff-patch` for unified diff generation instead of custom line comparison. The library uses the Myers diff algorithm to produce proper git-compatible diffs with context lines and hunk headers.

**Before implementing any algorithm or utility:**
1. Search pkg.go.dev for existing solutions
2. Check if go-git or other dependencies already provide the functionality
3. Only write custom code if no suitable library exists
- 3