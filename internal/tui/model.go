// Package tui provides the terminal user interface using Bubble Tea.
// It displays real-time progress of review modes, shows results with issues
// and suggestions, and handles user confirmation for commits.
package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/buker/revi/internal/review"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// State represents the current phase of the review and commit workflow.
type State int

const (
	StateAnalyzing        State = iota // Analyzing the diff to detect relevant review modes
	StateReviewing                     // Running code reviews in parallel
	StateBlocking                      // Blocked due to high-severity issues
	StateGeneratingCommit              // Generating the commit message
	StateConfirming                    // Waiting for user confirmation to commit
	StateDone                          // Workflow completed
	StateError                         // An error occurred
)

// Model is the main Bubble Tea model that manages the TUI state and rendering.
// It tracks the current workflow phase, review progress, results, and user interactions.
type Model struct {
	state           State                         // Current workflow phase
	modes           []review.Mode                 // Review modes to execute
	modeStatus      map[review.Mode]review.Status // Status of each review mode
	results         []*review.Result              // Collected review results
	commitMessage   string                        // Generated commit message
	error           string                        // Error message if in error state
	width           int                           // Terminal width
	height          int                           // Terminal height
	reasoning       string                        // Explanation of detected modes
	confirmed       bool                          // Whether user confirmed the commit
	blocked         bool                          // Whether commit was blocked
	blockReason     string                        // Reason for blocking
	resultsBuilder  strings.Builder               // Buffer for building results display
	reviewsComplete int                           // Count of completed reviews
	reviewsTotal    int                           // Total number of reviews to run
	mu              sync.RWMutex                  // Protects fields accessed across goroutines
	viewport        viewport.Model                // Scrollable viewport for results
	viewportReady   bool                          // Whether viewport has been initialized
}

// NewModel creates a new Model initialized to the analyzing state with empty collections.
func NewModel() *Model {
	return &Model{
		state:      StateAnalyzing,
		modeStatus: make(map[review.Mode]review.Status),
		results:    make([]*review.Result, 0),
	}
}

// Messages for updating the TUI from outside

// MsgModesDetected is sent when review modes are detected
type MsgModesDetected struct {
	Modes     []review.Mode
	Reasoning string
}

// MsgReviewStarted is sent when a review starts
type MsgReviewStarted struct {
	Mode review.Mode
}

// MsgReviewComplete is sent when a review completes
type MsgReviewComplete struct {
	Result *review.Result
}

// MsgAllReviewsComplete is sent when all reviews are done
type MsgAllReviewsComplete struct {
	Results []*review.Result
	Blocked bool
	Reason  string
}

// MsgCommitGenerated is sent when commit message is generated
type MsgCommitGenerated struct {
	Message string
}

// MsgError is sent when an error occurs
type MsgError struct {
	Error string
}

// MsgConfirm is sent to prompt for confirmation
type MsgConfirm struct{}

// MsgQuit is sent to quit the application
type MsgQuit struct{}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for header (3 lines) and footer (2 lines)
		headerHeight := 3
		footerHeight := 2
		viewportHeight := msg.Height - headerHeight - footerHeight
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		if !m.viewportReady {
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.viewport.YPosition = headerHeight
			m.viewportReady = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "y", "Y":
			if m.state == StateConfirming {
				m.mu.Lock()
				m.confirmed = true
				m.mu.Unlock()
				m.state = StateDone
				return m, tea.Quit
			}
		case "n", "N", "enter":
			if m.state == StateConfirming {
				m.mu.Lock()
				m.confirmed = false
				m.mu.Unlock()
				m.state = StateDone
				return m, tea.Quit
			}
		}

		// Forward key messages to viewport for scrolling (up, down, pgup, pgdown, etc.)
		if m.viewportReady {
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case MsgModesDetected:
		m.state = StateReviewing
		m.modes = msg.Modes
		m.reasoning = msg.Reasoning
		m.reviewsTotal = len(msg.Modes)
		for _, mode := range msg.Modes {
			m.modeStatus[mode] = review.StatusPending
		}
		return m, nil

	case MsgReviewStarted:
		m.modeStatus[msg.Mode] = review.StatusRunning
		return m, nil

	case MsgReviewComplete:
		if msg.Result != nil {
			m.modeStatus[msg.Result.Mode] = msg.Result.Status
			m.results = append(m.results, msg.Result)
			m.reviewsComplete++
			m.appendResult(msg.Result)
		}
		return m, nil

	case MsgAllReviewsComplete:
		m.results = msg.Results
		if msg.Blocked {
			m.state = StateBlocking
			m.mu.Lock()
			m.blocked = true
			m.mu.Unlock()
			m.blockReason = msg.Reason
		} else {
			m.state = StateGeneratingCommit
		}
		return m, nil

	case MsgCommitGenerated:
		m.mu.Lock()
		m.commitMessage = msg.Message
		m.mu.Unlock()
		m.state = StateConfirming
		return m, nil

	case MsgError:
		m.state = StateError
		m.error = msg.Error
		return m, tea.Quit

	case MsgQuit:
		return m, tea.Quit
	}

	return m, nil
}

// View renders the model
func (m *Model) View() string {
	var b strings.Builder

	// Fixed header
	b.WriteString("revi - AI Code Review & Commit\n")
	b.WriteString(strings.Repeat("-", 40) + "\n\n")

	// Build scrollable content
	var content strings.Builder

	// Progress section
	content.WriteString(m.renderProgress())
	content.WriteString("\n")

	// Results section
	if m.resultsBuilder.Len() > 0 {
		content.WriteString(m.resultsBuilder.String())
	}

	// State-specific content
	switch m.state {
	case StateAnalyzing:
		content.WriteString("Analyzing diff...\n")

	case StateBlocking:
		content.WriteString(strings.Repeat("-", 40) + "\n")
		content.WriteString("BLOCKED: " + m.blockReason + "\n")
		content.WriteString("Use --no-block to override\n")

	case StateGeneratingCommit:
		content.WriteString("\nGenerating commit message...\n")

	case StateConfirming:
		content.WriteString(strings.Repeat("-", 40) + "\n")
		content.WriteString("Commit message:\n\n")
		content.WriteString("  " + strings.ReplaceAll(m.commitMessage, "\n", "\n  ") + "\n\n")
		content.WriteString("Proceed with commit? [y/N] ")

	case StateError:
		content.WriteString("\nError: " + m.error + "\n")

	case StateDone:
		if m.confirmed {
			content.WriteString("\nCommit created.\n")
		} else if m.blocked {
			content.WriteString("\nCommit blocked.\n")
		} else {
			content.WriteString("\nCommit cancelled.\n")
		}
	}

	// Render viewport with content or fallback to plain text
	if m.viewportReady {
		m.viewport.SetContent(content.String())
		b.WriteString(m.viewport.View())

		// Footer with scroll info
		b.WriteString("\n")
		scrollPercent := m.viewport.ScrollPercent() * 100
		b.WriteString(fmt.Sprintf("↑/↓ scroll • %.0f%% • q quit", scrollPercent))
	} else {
		b.WriteString(content.String())
	}

	return b.String()
}

// renderProgress renders the progress header
func (m *Model) renderProgress() string {
	var b strings.Builder

	if m.state == StateAnalyzing {
		b.WriteString("Status: Analyzing...\n")
		return b.String()
	}

	// Show reasoning if available
	if m.reasoning != "" {
		b.WriteString(fmt.Sprintf("Detected: %s\n\n", m.reasoning))
	}

	// Reviews progress
	b.WriteString(fmt.Sprintf("Reviews: %d/%d\n", m.reviewsComplete, m.reviewsTotal))
	b.WriteString(strings.Repeat("-", 30) + "\n")

	// Individual mode status
	for _, mode := range m.modes {
		status := m.modeStatus[mode]
		info := review.GetModeInfo(mode)
		statusStr := m.statusToString(status)
		b.WriteString(fmt.Sprintf("%-15s %s\n", info.Name+":", statusStr))
	}

	return b.String()
}

// statusToString converts a status to a display string
func (m *Model) statusToString(status review.Status) string {
	switch status {
	case review.StatusPending:
		return "Pending"
	case review.StatusRunning:
		return "Running..."
	case review.StatusDone, review.StatusNoIssues:
		return "Done"
	case review.StatusIssues:
		return "Done (issues found)"
	case review.StatusFailed:
		return "Failed"
	default:
		return string(status)
	}
}

// appendResult appends a review result to the results display
func (m *Model) appendResult(r *review.Result) {
	info := review.GetModeInfo(r.Mode)
	m.resultsBuilder.WriteString("\n")
	m.resultsBuilder.WriteString(fmt.Sprintf("=== %s Review ===\n", info.Name))

	if r.Status == review.StatusFailed {
		m.resultsBuilder.WriteString(fmt.Sprintf("Status: Failed (%s)\n", r.Error))
		return
	}

	if len(r.Issues) == 0 {
		m.resultsBuilder.WriteString("Status: No issues found\n")
	} else {
		m.resultsBuilder.WriteString(fmt.Sprintf("Status: %d issue(s) found\n", len(r.Issues)))
	}

	if r.Summary != "" {
		m.resultsBuilder.WriteString(fmt.Sprintf("\nSummary:\n  %s\n", r.Summary))
	}

	if len(r.Issues) > 0 {
		m.resultsBuilder.WriteString("\nIssues:\n")
		for _, issue := range r.Issues {
			loc := ""
			if issue.Location != "" {
				loc = fmt.Sprintf(" (%s)", issue.Location)
			}
			m.resultsBuilder.WriteString(fmt.Sprintf("  - [%s] %s%s\n",
				strings.ToUpper(issue.Severity), issue.Description, loc))
		}
	}

	if len(r.Suggestions) > 0 {
		m.resultsBuilder.WriteString("\nSuggestions:\n")
		for _, s := range r.Suggestions {
			m.resultsBuilder.WriteString(fmt.Sprintf("  - %s\n", s))
		}
	}
}

// IsConfirmed returns whether the user confirmed the commit
func (m *Model) IsConfirmed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.confirmed
}

// IsBlocked returns whether the commit was blocked
func (m *Model) IsBlocked() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.blocked
}

// GetCommitMessage returns the generated commit message
func (m *Model) GetCommitMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.commitMessage
}
