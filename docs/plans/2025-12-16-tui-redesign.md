# TUI Redesign: k9s-Style Compact Interface

## Overview

Redesign the revi TUI from a scrolling text view to a compact, k9s-style table-based interface with modal overlays for details and fix application.

## Design Decisions

- **Issues table columns**: Severity | Mode | Location | Summary | Has Fix (5 columns)
- **Detail view style**: Modal overlay (Esc to close)
- **Fix application**: Preview diff first, confirm with 'y' to apply
- **Review progress**: Table with Mode | Status | Duration | Issues
- **Keybindings**: Both vim (j/k) and arrow keys supported

## Architecture

### Views

```
┌─────────────────────────────────────────────────────┐
│  1. ReviewProgressView (during analysis/reviews)    │
│     - Shows while reviews run                       │
│     - Transitions to IssuesTableView when complete  │
├─────────────────────────────────────────────────────┤
│  2. IssuesTableView (main interactive screen)       │
│     - Compact table of all issues                   │
│     - Navigate with j/k or arrows                   │
│     - Enter opens detail modal                      │
├─────────────────────────────────────────────────────┤
│  3. IssueDetailModal (overlay on IssuesTableView)   │
│     - Full issue description                        │
│     - Fix preview with diff                         │
│     - Apply fix action                              │
├─────────────────────────────────────────────────────┤
│  4. DiffPreviewModal (overlay for fix preview)      │
│     - Shows unified diff of proposed fix            │
│     - Confirm or cancel application                 │
├─────────────────────────────────────────────────────┤
│  5. CommitConfirmView (final confirmation)          │
│     - Shows commit message                          │
│     - Review summary with fix count                 │
│     - Edit message option                           │
└─────────────────────────────────────────────────────┘
```

### State Transitions

```
Analyzing → ReviewProgress → IssuesTable → CommitConfirm → Done
                                 │
                                 ├── [Enter] → IssueDetailModal
                                 │                  │
                                 │                  └── [a] → DiffPreviewModal
                                 │                              │
                                 │                              └── [y] → Apply fix, return
                                 │
                                 └── [c] → CommitConfirm
```

### File Structure

```
internal/tui/
├── model.go           # Main model, state machine, message types
├── program.go         # Program wrapper (existing, minor updates)
├── styles.go          # Shared lipgloss styles
├── keys.go            # Keybinding definitions
└── views/
    ├── progress.go    # ReviewProgressView
    ├── issues.go      # IssuesTableView
    ├── detail.go      # IssueDetailModal
    ├── diff.go        # DiffPreviewModal
    └── commit.go      # CommitConfirmView
```

## View Specifications

### 1. ReviewProgressView

```
revi - AI Code Review
──────────────────────────────────────────────────────
 MODE           │ STATUS      │ DURATION │ ISSUES
──────────────────────────────────────────────────────
 Security       │ ✓ Done      │ 2.3s     │ 2
 Performance    │ ◐ Running   │ 1.5s     │ -
 Style          │ ○ Pending   │ -        │ -
 Error Handling │ ✓ Done      │ 1.8s     │ 0
──────────────────────────────────────────────────────
 Progress: 2/4 complete

 [q] quit
```

**Status indicators:**
- `○` pending
- `◐` running (animated spinner)
- `✓` done
- `✗` failed

**Behavior:**
- Non-interactive (no row selection)
- Duration updates live for running reviews
- Auto-transitions to IssuesTableView when complete

### 2. IssuesTableView

```
revi - Issues (3 found)                    [2/3]
──────────────────────────────────────────────────────
 SEV  │ MODE        │ LOCATION       │ SUMMARY                          │ FIX
──────────────────────────────────────────────────────
 HIG  │ Security    │ api/auth.go:42 │ SQL injection in query param     │ ✓
▶MED  │ Performance │ db/repo.go:87  │ N+1 query in loop                │ ✓
 LOW  │ Style       │ cmd/main.go:15 │ Unexported func missing comment  │ ✗
──────────────────────────────────────────────────────

 Commit: feat(api): add user authentication endpoint
──────────────────────────────────────────────────────
 [↑/k] up  [↓/j] down  [Enter] details  [c] commit  [q] quit
```

**Columns:**
- SEV: HIG/MED/LOW with color coding (red/yellow/gray)
- MODE: Review mode name
- LOCATION: file:line
- SUMMARY: Truncated issue description
- FIX: ✓ if available, ✗ if not

**Selection:**
- `▶` marker on current row
- Row highlighted

**Keybindings:**
- `j` / `↓` - move down
- `k` / `↑` - move up
- `Enter` - open detail modal
- `c` - proceed to commit
- `q` - quit

### 3. IssueDetailModal

```
┌─────────────────────────────────────────────────────┐
│ Performance Issue                                   │
│─────────────────────────────────────────────────────│
│ Location: db/repo.go:87                             │
│ Severity: MEDIUM                                    │
│                                                     │
│ Description:                                        │
│ N+1 query detected inside loop. Each iteration      │
│ executes a separate DB query. This will cause       │
│ performance issues at scale with O(n) database      │
│ round trips.                                        │
│                                                     │
│ Fix available: Yes                                  │
│ Explanation: Batch the queries using a single IN    │
│ clause instead of loop.                             │
│                                                     │
│─────────────────────────────────────────────────────│
│ [a] preview fix  [Esc] close                        │
└─────────────────────────────────────────────────────┘
```

**Content:**
- Mode/type in header
- Location and severity
- Full description (scrollable)
- Fix availability and explanation

**Keybindings:**
- `Esc` - close modal
- `a` - open diff preview (if fix available)
- `↑/↓/j/k` - scroll content

### 4. DiffPreviewModal

```
┌─────────────────────────────────────────────────────┐
│ Fix Preview: db/repo.go                             │
│─────────────────────────────────────────────────────│
│ @@ -85,7 +85,10 @@                                  │
│                                                     │
│  func (r *Repo) GetUsers(ids []int) ([]User, error) │
│-     var users []User                               │
│-     for _, id := range ids {                       │
│-         u, _ := r.db.Query("SELECT * FROM users    │
│-                             WHERE id = ?", id)     │
│-         users = append(users, u)                   │
│-     }                                              │
│+     users, err := r.db.Query(                      │
│+         "SELECT * FROM users WHERE id IN (?)",    │
│+         ids,                                       │
│+     )                                              │
│+     if err != nil {                                │
│+         return nil, err                            │
│+     }                                              │
│      return users, nil                              │
│                                                     │
│─────────────────────────────────────────────────────│
│ [y] apply fix  [n/Esc] cancel                       │
└─────────────────────────────────────────────────────┘
```

**Content:**
- File path in header
- Unified diff with colors (red removed, green added)
- Scrollable for large diffs

**Keybindings:**
- `y` - apply fix to file
- `n` / `Esc` - cancel

**After apply:**
- Close modal
- Update issue status to "Fixed" in table
- Show [FIXED] indicator on row

### 5. CommitConfirmView

```
revi - Confirm Commit
──────────────────────────────────────────────────────

 Commit Message:
 ┌──────────────────────────────────────────────────┐
 │ feat(api): add user authentication endpoint      │
 │                                                  │
 │ - Implement JWT token validation                 │
 │ - Add login/logout handlers                      │
 │ - Create auth middleware                         │
 └──────────────────────────────────────────────────┘

 Review Summary:
 ────────────────────────────────────────────────────
  Issues: 3 found (1 fixed, 2 remaining)
  Blocked: No

──────────────────────────────────────────────────────
 [y] commit  [e] edit message  [n/Esc] cancel
```

**Keybindings:**
- `y` - create commit, exit
- `e` - edit commit message inline
- `n` / `Esc` - return to issues table

## Styling

Use lipgloss for consistent styling:

```go
var (
    // Colors
    ColorHigh   = lipgloss.Color("#FF5555")  // Red
    ColorMedium = lipgloss.Color("#FFAA00")  // Yellow/Orange
    ColorLow    = lipgloss.Color("#888888")  // Gray
    ColorGreen  = lipgloss.Color("#55FF55")  // Green (for additions)
    ColorBorder = lipgloss.Color("#444444")  // Border color

    // Styles
    TableHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
    SelectedRow = lipgloss.NewStyle().Background(lipgloss.Color("#333333"))
    ModalBox    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)
)
```

## Implementation Notes

### Fix Application

The fix application requires:
1. Read the target file
2. Replace lines StartLine to EndLine with Fix.Code
3. Write the file back
4. Track which issues have been fixed in the model

### Modal Rendering

Modals should:
1. Calculate center position based on terminal size
2. Render the background (dimmed table)
3. Render the modal box on top
4. Handle focus for keybindings

### Spinner Animation

Use bubbles/spinner for the running status indicator with a tick command to animate.

## Dependencies

Existing (no new deps needed):
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - Components (viewport, spinner)
- `github.com/charmbracelet/lipgloss` - Styling
