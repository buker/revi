# revi

AI-powered code review and commit message generator using Claude.

## Overview

revi analyzes your staged git changes and provides specialized code reviews before generating conventional commit messages. It integrates with the Claude CLI to provide intelligent feedback on security, performance, style, error handling, testing, and documentation.

## Features

- **Intelligent Code Review**: Automatically detects relevant review modes based on your changes
- **Specialized Review Modes**:
  - Security: SQL injection, XSS, authentication issues, secrets exposure
  - Performance: N+1 queries, unnecessary loops, caching opportunities
  - Style: Naming conventions, code patterns, consistency
  - Errors: Missing error checks, swallowed exceptions, edge cases
  - Testing: Untested code paths, missing assertions, coverage gaps
  - Docs: Missing comments, unclear names, API documentation
- **Commit Message Generation**: Creates conventional commit messages (feat, fix, docs, etc.)
- **Interactive TUI**: Real-time progress display with review results
- **Configurable**: Per-project or global configuration via YAML

## Prerequisites

- [Claude CLI](https://claude.ai/download) installed and authenticated
- Git repository with staged changes

## Installation

```bash
go install github.com/buker/revi/cmd/revi@latest
```

Or build from source:

```bash
git clone https://github.com/buker/revi.git
cd revi
make build
```

## Usage

### Full Workflow

Stage your changes and run revi:

```bash
git add .
revi
```

This will:
1. Analyze staged changes to detect relevant review modes
2. Run specialized reviews in parallel
3. Generate a commit message
4. Prompt for confirmation before committing

### Review Only

Run code review without committing:

```bash
revi review
```

### Generate Commit Message Only

Generate a commit message without review:

```bash
revi commit
```

### Command Line Options

```bash
# Disable code review
revi --no-review

# Don't block on high-severity issues
revi --no-block

# Preview without committing
revi --dry-run

# Run all review modes
revi --all

# Enable/disable specific modes
revi --security --no-style
revi --performance --testing

# Show version
revi version
```

## Configuration

Create `.revi.yaml` in your project root or `~/.revi.yaml` for global settings:

```yaml
review:
  enabled: true
  block: true  # Block commit on high-severity issues
  modes:
    security: true
    performance: true
    style: true
    errors: true
    testing: true
    docs: true

commit:
  enabled: true

claude:
  path: "claude"  # Path to Claude CLI
  timeout: 60     # Timeout in seconds
```

Environment variables are also supported with the `REVI_` prefix:

```bash
export REVI_CLAUDE_PATH=/usr/local/bin/claude
export REVI_CLAUDE_TIMEOUT=120
export REVI_REVIEW_BLOCK=false
```

## How It Works

1. **Mode Detection**: revi analyzes your diff using Claude to determine which review modes are relevant. Falls back to heuristic detection if needed.

2. **Parallel Reviews**: Selected review modes run concurrently, each focused on its specific concerns.

3. **Issue Reporting**: Issues are categorized by severity (high/medium/low) with locations and actionable suggestions.

4. **Blocking**: By default, high-severity issues block the commit. Use `--no-block` to override.

5. **Commit Generation**: Claude generates a conventional commit message based on the actual changes.

## Project Structure

```
cmd/revi/          # Application entry point
internal/
  claude/          # Claude CLI client
  cli/             # Command-line interface (cobra)
  commit/          # Commit message generation
  config/          # Configuration management (viper)
  git/             # Git operations (go-git)
  review/          # Review modes, detection, and execution
  tui/             # Terminal UI (bubble tea)
```

## License

MIT
