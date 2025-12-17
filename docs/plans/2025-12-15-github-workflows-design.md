# GitHub Workflows Design

## Overview

This document describes the CI and release workflows for the revi project.

## CI Workflow

**File:** `.github/workflows/ci.yml`

**Triggers:**
- Push to `main` branch
- Pull requests targeting `main`

**Jobs:**

### lint
- Uses `golangci/golangci-lint-action`
- Runs golangci-lint on all packages

### test
- Uses `actions/setup-go@v5` with Go 1.25
- Runs `go test -v ./...`

Both jobs run in parallel for faster feedback.

## Release Workflow

**File:** `.github/workflows/release.yml`

**Trigger:** Manual workflow dispatch with version bump selector

**Input:**
```yaml
version_bump:
  description: 'Version bump type'
  required: true
  type: choice
  options:
    - patch
    - minor
    - major
```

**Steps:**
1. Checkout code with full git history
2. Get latest tag (e.g., `v1.2.3`)
3. Calculate new version based on bump type
4. Create and push new tag
5. Run GoReleaser to build and publish release

## GoReleaser Configuration

**File:** `.goreleaser.yaml`

**Build targets:**
| OS      | Architectures    |
|---------|------------------|
| Linux   | amd64, arm64     |
| macOS   | amd64, arm64     |
| Windows | amd64, arm64     |

**Archive formats:**
- Linux/macOS: `.tar.gz`
- Windows: `.zip`

**Naming:** `revi_<version>_<os>_<arch>.<ext>`

**Checksums:** SHA256 in `checksums.txt`

**Changelog:** Auto-generated from commits, excluding `docs:`, `test:`, `chore:` prefixes

**ldflags:** Embeds version, commit, and build time (matching Makefile pattern)

## Required Setup

### GitHub Repository
- `GITHUB_TOKEN` - automatically available
- Workflow permissions: `contents: write` for creating tags/releases

### Initial Setup
1. Create initial tag if none exists (e.g., `v0.0.0`)
2. Optionally install GoReleaser locally for testing

### Local Testing
```bash
goreleaser check                      # validate config
goreleaser build --snapshot --clean   # test build
```

## Files to Create

1. `.github/workflows/ci.yml`
2. `.github/workflows/release.yml`
3. `.goreleaser.yaml`
