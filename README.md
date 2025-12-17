# revi

AI-powered code review and commit message generator using Claude.

## Overview

revi analyzes your staged git changes and provides specialized code reviews before generating conventional commit messages. It uses the Claude Code SDK to communicate with Claude models through a persistent subprocess connection for intelligent feedback on security, performance, style, error handling, testing, and documentation.

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
- **Streaming Responses**: See AI output in real-time as reviews progress
- **Configurable**: Per-project or global configuration via YAML

## Prerequisites

- Go 1.21 or later
- Git repository with staged changes
- Node.js and npm (for Claude Code CLI installation)
- Claude Code CLI (`@anthropic-ai/claude-code`)
- Active Claude Max subscription (required for authentication)

## Installation

### 1. Install Claude Code CLI

First, install the Claude Code CLI globally using npm:

```bash
npm install -g @anthropic-ai/claude-code
```

Verify the installation:

```bash
claude --version
```

### 2. Authenticate with Claude

Authenticate using your Claude Max subscription credentials:

```bash
claude login
```

This will open a browser window for authentication. Once complete, you can verify authentication:

```bash
claude auth check
```

### 3. Install revi

Install revi using Go:

```bash
go install github.com/buker/revi/cmd/revi@latest
```

Or build from source:

```bash
git clone https://github.com/buker/revi.git
cd revi
make build
```

## Authentication

revi uses the Claude Code CLI for authentication, which delegates to your Claude Max subscription. Authentication is handled automatically once you've completed the `claude login` step during installation.

**No additional environment variables or API tokens are required.** The Claude Code CLI manages authentication internally through a secure subprocess connection.

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

# Use a different AI model
revi --model claude-sonnet-4-20250514

# Enable debug logging
revi --debug

# Show version
revi version
```

### Model Selection

You can override the default model (Claude Opus 4.5) using:

- Command-line flag: `--model claude-sonnet-4-20250514`
- Environment variable: `export REVI_AI_MODEL=claude-sonnet-4-20250514`
- Config file: Set `ai.model` in `.revi.yaml`

Available models include:
- `claude-opus-4-5-20251101` (default, most capable)
- `claude-sonnet-4-20250514` (balanced performance/cost)
- `claude-3-5-haiku-20241022` (fastest, lowest cost)

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

ai:
  model: "claude-opus-4-5-20251101"  # AI model to use
```

Environment variables are also supported with the `REVI_` prefix:

```bash
export REVI_AI_MODEL=claude-sonnet-4-20250514
export REVI_REVIEW_BLOCK=false
```

## Troubleshooting

### Authentication Errors

If you encounter authentication errors:

1. Verify Claude Code CLI is installed: `claude --version`
2. Check authentication status: `claude auth check`
3. Re-authenticate if needed: `claude login`
4. Ensure you have an active Claude Max subscription

### Rate Limiting

revi implements automatic retry with exponential backoff for rate limits. If you see persistent rate limit errors:

1. Wait a few minutes before retrying
2. Consider using a less intensive model (e.g., Haiku)
3. Reduce the number of review modes with `--no-style --no-docs`

### Network Errors

revi automatically retries network errors once. If issues persist:

1. Check your internet connection
2. Verify the Claude Code CLI is functioning: `claude --version`
3. Try again in a few minutes

## How It Works

1. **Mode Detection**: revi analyzes your diff using Claude to determine which review modes are relevant. Falls back to heuristic detection if needed.

2. **Parallel Reviews**: Selected review modes run concurrently, each focused on its specific concerns.

3. **Streaming Output**: Review progress displays in real-time as Claude processes your code.

4. **Issue Reporting**: Issues are categorized by severity (high/medium/low) with locations and actionable suggestions.

5. **Blocking**: By default, high-severity issues block the commit. Use `--no-block` to override.

6. **Commit Generation**: Claude generates a conventional commit message based on the actual changes.

## Project Structure

```
cmd/revi/          # Application entry point
internal/
  ai/              # Claude Code SDK client
  cli/             # Command-line interface (cobra)
  commit/          # Commit message generation
  config/          # Configuration management (viper)
  git/             # Git operations (go-git)
  review/          # Review modes, detection, and execution
  tui/             # Terminal UI (bubble tea)
```

## License

MIT
