// Package tui provides the terminal user interface using Bubble Tea.
// It displays real-time progress of review modes, shows results with issues
// and suggestions, and handles user confirmation for commits.
package tui

import (
	"sync"

	"github.com/buker/revi/internal/review"
	"github.com/buker/revi/internal/tui/views"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// State represents the current phase of the review and commit workflow.
type State int

const (
	StateAnalyzing        State = iota // Analyzing the diff to detect relevant review modes
	StateReviewing                     // Running code reviews in parallel
	StateIssuesTable                   // Showing issues table (main interactive screen)
	StateIssueDetail                   // Showing issue detail modal
	StateDiffPreview                   // Showing diff preview modal
	StateCommitConfirm                 // Commit confirmation screen
	StateBlocking                      // Blocked due to high-severity issues
	StateDone                          // Workflow completed
	StateError                         // An error occurred
)

// FixApplier is a function that applies a fix and returns an error if it fails
type FixApplier func(*review.Fix) error

// Model is the main Bubble Tea model that manages the TUI state and rendering.
type Model struct {
	state   State  // Current workflow phase
	width   int    // Terminal width
	height  int    // Terminal height
	error   string // Error message if in error state

	// Results
	results       []*review.Result // Collected review results
	commitMessage string           // Generated commit message
	confirmed     bool             // Whether user confirmed the commit
	blocked       bool             // Whether commit was blocked
	blockReason   string           // Reason for blocking

	// Fix tracking
	fixedIssues map[int]bool // Track which issues have been fixed (by index)
	fixApplier  FixApplier   // Callback for applying fixes

	// View components
	progressView *views.ProgressView
	issuesView   *views.IssuesTableView
	detailModal  *views.IssueDetailModal
	diffModal    *views.DiffPreviewModal
	commitView   *views.CommitConfirmView

	// Keybindings
	keys KeyMap

	// Thread safety
	mu sync.RWMutex
}

// NewModel creates a new Model initialized to the analyzing state.
func NewModel() *Model {
	return &Model{
		state:        StateAnalyzing,
		progressView: views.NewProgressView(),
		issuesView:   views.NewIssuesTableView(),
		detailModal:  views.NewIssueDetailModal(),
		diffModal:    views.NewDiffPreviewModal(),
		commitView:   views.NewCommitConfirmView(),
		keys:         DefaultKeyMap(),
		fixedIssues:  make(map[int]bool),
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

// MsgFixApplied is sent when a fix has been applied
type MsgFixApplied struct {
	IssueIndex int
	Success    bool
	Error      string
}

// MsgQuit is sent to quit the application
type MsgQuit struct{}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return m.progressView.Init()
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progressView.SetSize(msg.Width, msg.Height)
		m.issuesView.SetSize(msg.Width, msg.Height)
		m.detailModal.SetSize(msg.Width, msg.Height)
		m.diffModal.SetSize(msg.Width, msg.Height)
		m.commitView.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case MsgModesDetected:
		m.state = StateReviewing
		m.progressView.SetModes(msg.Modes)
		return m, nil

	case MsgReviewStarted:
		m.progressView.SetReviewStarted(msg.Mode)
		return m, nil

	case MsgReviewComplete:
		if msg.Result != nil {
			m.progressView.SetReviewComplete(msg.Result.Mode, msg.Result.Status, len(msg.Result.Issues))
		}
		// Keep spinner ticking
		pv, cmd := m.progressView.Update(msg)
		m.progressView = pv
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case MsgAllReviewsComplete:
		m.results = msg.Results
		m.issuesView.SetIssues(msg.Results)
		if msg.Blocked {
			m.state = StateBlocking
			m.mu.Lock()
			m.blocked = true
			m.mu.Unlock()
			m.blockReason = msg.Reason
		} else {
			m.state = StateIssuesTable
		}
		return m, nil

	case MsgCommitGenerated:
		m.mu.Lock()
		m.commitMessage = msg.Message
		m.mu.Unlock()
		m.issuesView.SetCommitMessage(msg.Message)
		m.commitView.SetCommitMessage(msg.Message)
		return m, nil

	case MsgError:
		m.state = StateError
		m.error = msg.Error
		return m, tea.Quit

	case MsgFixApplied:
		if msg.Success {
			m.fixedIssues[msg.IssueIndex] = true
			m.issuesView.MarkFixed(msg.IssueIndex)
		}
		// Return to issues table after fix
		m.state = StateIssuesTable
		return m, nil

	case MsgQuit:
		return m, tea.Quit
	}

	// Update spinner for progress view
	if m.state == StateReviewing {
		pv, cmd := m.progressView.Update(msg)
		m.progressView = pv
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input based on current state
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}

	switch m.state {
	case StateReviewing:
		// No interactive keys during review, just allow quit
		return m, nil

	case StateIssuesTable:
		return m.handleIssuesTableKeys(msg)

	case StateIssueDetail:
		return m.handleIssueDetailKeys(msg)

	case StateDiffPreview:
		return m.handleDiffPreviewKeys(msg)

	case StateCommitConfirm:
		return m.handleCommitConfirmKeys(msg)

	case StateBlocking:
		// Just allow quit
		return m, nil
	}

	return m, nil
}

// handleIssuesTableKeys handles keys in the issues table view
func (m *Model) handleIssuesTableKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down),
		key.Matches(msg, m.keys.Home), key.Matches(msg, m.keys.End):
		iv, cmd := m.issuesView.Update(msg)
		m.issuesView = iv
		return m, cmd

	case key.Matches(msg, m.keys.Enter):
		// Open issue detail modal
		if item := m.issuesView.SelectedIssue(); item != nil {
			m.detailModal.SetIssue(&item.Issue, item.Mode)
			m.detailModal.SetSize(m.width, m.height)
			m.state = StateIssueDetail
		}
		return m, nil

	case key.Matches(msg, m.keys.Commit):
		// Go to commit confirm
		m.updateCommitSummary()
		m.state = StateCommitConfirm
		return m, nil
	}

	return m, nil
}

// handleIssueDetailKeys handles keys in the issue detail modal
func (m *Model) handleIssueDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		// Close modal, return to issues table
		m.state = StateIssuesTable
		return m, nil

	case key.Matches(msg, m.keys.Apply):
		// Open diff preview if fix available
		if m.detailModal.HasFix() {
			if item := m.issuesView.SelectedIssue(); item != nil && item.Issue.Fix != nil {
				m.diffModal.SetFix(item.Issue.Fix)
				m.diffModal.SetSize(m.width, m.height)
				m.state = StateDiffPreview
			}
		}
		return m, nil

	default:
		// Pass to modal for scrolling
		dm, cmd := m.detailModal.Update(msg)
		m.detailModal = dm
		return m, cmd
	}
}

// handleDiffPreviewKeys handles keys in the diff preview modal
func (m *Model) handleDiffPreviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Cancel):
		// Close modal, return to issue detail
		m.state = StateIssueDetail
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		// Apply fix using the fix applier callback
		fix := m.diffModal.GetFix()
		issueIdx := m.issuesView.Cursor()

		if fix == nil || m.fixApplier == nil {
			// No fix or applier, just return to issues
			m.state = StateIssuesTable
			return m, nil
		}

		// Return a command that applies the fix asynchronously
		return m, func() tea.Msg {
			err := m.fixApplier(fix)
			if err != nil {
				return MsgFixApplied{
					IssueIndex: issueIdx,
					Success:    false,
					Error:      err.Error(),
				}
			}
			return MsgFixApplied{
				IssueIndex: issueIdx,
				Success:    true,
			}
		}

	default:
		// Pass to modal for scrolling
		dm, cmd := m.diffModal.Update(msg)
		m.diffModal = dm
		return m, cmd
	}
}

// handleCommitConfirmKeys handles keys in the commit confirm view
func (m *Model) handleCommitConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If editing, handle textarea
	if m.commitView.IsEditing() {
		switch msg.String() {
		case "esc":
			m.commitView.CancelEditing()
			return m, nil
		case "ctrl+d":
			m.commitView.StopEditing()
			m.commitMessage = m.commitView.GetCommitMessage()
			return m, nil
		default:
			cv, cmd := m.commitView.Update(msg)
			m.commitView = cv
			return m, cmd
		}
	}

	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Cancel):
		// Return to issues table
		m.state = StateIssuesTable
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		// Confirm commit
		m.mu.Lock()
		m.confirmed = true
		m.commitMessage = m.commitView.GetCommitMessage()
		m.mu.Unlock()
		m.state = StateDone
		return m, tea.Quit

	case key.Matches(msg, m.keys.Edit):
		// Enter edit mode
		return m, m.commitView.StartEditing()
	}

	return m, nil
}

// updateCommitSummary updates the commit view with current summary
func (m *Model) updateCommitSummary() {
	issuesFound := m.issuesView.IssueCount()
	issuesFixed := len(m.fixedIssues)
	m.commitView.SetReviewSummary(issuesFound, issuesFixed, m.blocked)
}

// View renders the model
func (m *Model) View() string {
	switch m.state {
	case StateAnalyzing:
		return m.renderAnalyzing()

	case StateReviewing:
		return m.progressView.View()

	case StateIssuesTable:
		return m.issuesView.View()

	case StateIssueDetail:
		// Render modal over issues table
		return m.detailModal.OverlayOnBackground(m.issuesView.View())

	case StateDiffPreview:
		return m.diffModal.View()

	case StateCommitConfirm:
		return m.commitView.View()

	case StateBlocking:
		return m.renderBlocked()

	case StateError:
		return m.renderError()

	case StateDone:
		return m.renderDone()
	}

	return ""
}

// renderAnalyzing renders the analyzing state
func (m *Model) renderAnalyzing() string {
	return TitleStyle.Render("revi - AI Code Review") + "\n" +
		RenderDivider(40) + "\n\n" +
		"Analyzing diff...\n\n" +
		HelpKeyStyle.Render(ProgressHelp())
}

// renderBlocked renders the blocked state
func (m *Model) renderBlocked() string {
	return TitleStyle.Render("revi - AI Code Review") + "\n" +
		RenderDivider(40) + "\n\n" +
		HighSeverityStyle.Render("BLOCKED: "+m.blockReason) + "\n\n" +
		"Use --no-block to override\n\n" +
		HelpKeyStyle.Render(" [q] quit")
}

// renderError renders the error state
func (m *Model) renderError() string {
	return TitleStyle.Render("revi - AI Code Review") + "\n" +
		RenderDivider(40) + "\n\n" +
		HighSeverityStyle.Render("Error: "+m.error) + "\n"
}

// renderDone renders the done state
func (m *Model) renderDone() string {
	msg := "Commit cancelled."
	if m.confirmed {
		msg = "Commit created."
	} else if m.blocked {
		msg = "Commit blocked."
	}
	return TitleStyle.Render("revi - AI Code Review") + "\n" +
		RenderDivider(40) + "\n\n" +
		msg + "\n"
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

// GetFixedIssues returns the set of fixed issue indices
func (m *Model) GetFixedIssues() map[int]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race conditions
	result := make(map[int]bool)
	for k, v := range m.fixedIssues {
		result[k] = v
	}
	return result
}

// GetSelectedFix returns the fix for the currently selected issue (for external application)
func (m *Model) GetSelectedFix() *review.Fix {
	if item := m.issuesView.SelectedIssue(); item != nil && item.Issue.Fix != nil {
		return item.Issue.Fix
	}
	return nil
}

// GetSelectedIssueIndex returns the index of the currently selected issue
func (m *Model) GetSelectedIssueIndex() int {
	return m.issuesView.Cursor()
}

// SetFixApplier sets the callback function for applying fixes
func (m *Model) SetFixApplier(applier FixApplier) {
	m.fixApplier = applier
}
