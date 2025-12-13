package tui

import (
	"context"

	"github.com/buker/revi/internal/review"
	tea "github.com/charmbracelet/bubbletea"
)

// Program wraps a Bubble Tea program to provide a higher-level API for external control.
// It allows other parts of the application to send state updates to the TUI while
// it runs in a separate goroutine.
type Program struct {
	program *tea.Program // Underlying Bubble Tea program
	model   *Model       // Shared model for state access
}

// NewProgram creates and initializes a new TUI Program ready to be started.
func NewProgram() *Program {
	model := NewModel()
	program := tea.NewProgram(model)
	return &Program{
		program: program,
		model:   model,
	}
}

// Start runs the TUI program and blocks until it exits.
// Returns an error if the program fails to initialize or encounters a fatal error.
func (p *Program) Start() error {
	_, err := p.program.Run()
	return err
}

// Send dispatches a message to the TUI for processing.
// This is thread-safe and can be called from any goroutine.
func (p *Program) Send(msg tea.Msg) {
	p.program.Send(msg)
}

// SetModesDetected notifies the TUI that modes have been detected
func (p *Program) SetModesDetected(modes []review.Mode, reasoning string) {
	p.Send(MsgModesDetected{Modes: modes, Reasoning: reasoning})
}

// SetReviewStarted notifies the TUI that a review has started
func (p *Program) SetReviewStarted(mode review.Mode) {
	p.Send(MsgReviewStarted{Mode: mode})
}

// SetReviewComplete notifies the TUI that a review has completed
func (p *Program) SetReviewComplete(result *review.Result) {
	p.Send(MsgReviewComplete{Result: result})
}

// SetAllReviewsComplete notifies the TUI that all reviews are done
func (p *Program) SetAllReviewsComplete(results []*review.Result, blocked bool, reason string) {
	p.Send(MsgAllReviewsComplete{Results: results, Blocked: blocked, Reason: reason})
}

// SetCommitGenerated notifies the TUI that a commit message was generated
func (p *Program) SetCommitGenerated(message string) {
	p.Send(MsgCommitGenerated{Message: message})
}

// SetError notifies the TUI of an error
func (p *Program) SetError(err string) {
	p.Send(MsgError{Error: err})
}

// Quit quits the TUI
func (p *Program) Quit() {
	p.Send(MsgQuit{})
}

// IsConfirmed returns whether the user confirmed the action
func (p *Program) IsConfirmed() bool {
	return p.model.IsConfirmed()
}

// IsBlocked returns whether the commit was blocked
func (p *Program) IsBlocked() bool {
	return p.model.IsBlocked()
}

// GetCommitMessage returns the generated commit message
func (p *Program) GetCommitMessage() string {
	return p.model.GetCommitMessage()
}

// RunWithCallbacks orchestrates the complete review workflow with real-time TUI updates.
// It starts the TUI in a background goroutine, then executes mode detection, parallel reviews,
// and commit message generation, updating the TUI at each step. Returns when the TUI exits.
func (p *Program) RunWithCallbacks(
	ctx context.Context,
	detectFunc func(ctx context.Context) ([]review.Mode, string, error),
	reviewFunc func(ctx context.Context, mode review.Mode) (*review.Result, error),
	commitFunc func(ctx context.Context) (string, error),
	blockOnIssues bool,
) error {
	// Run TUI in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- p.Start()
	}()

	// Detect modes
	modes, reasoning, err := detectFunc(ctx)
	if err != nil {
		p.SetError(err.Error())
		return <-errCh
	}
	p.SetModesDetected(modes, reasoning)

	// Run reviews in parallel
	results := make([]*review.Result, len(modes))
	resultsCh := make(chan struct {
		idx    int
		result *review.Result
	}, len(modes))

	for i, mode := range modes {
		go func(idx int, m review.Mode) {
			p.SetReviewStarted(m)
			result, err := reviewFunc(ctx, m)
			if err != nil {
				result = &review.Result{
					Mode:   m,
					Status: review.StatusFailed,
					Error:  err.Error(),
				}
			}
			p.SetReviewComplete(result)
			resultsCh <- struct {
				idx    int
				result *review.Result
			}{idx, result}
		}(i, mode)
	}

	// Collect results
	for range modes {
		r := <-resultsCh
		results[r.idx] = r.result
	}

	// Check if should block
	blocked := review.ShouldBlock(results, blockOnIssues)
	blockReason := review.GetBlockReason(results)
	p.SetAllReviewsComplete(results, blocked, blockReason)

	if blocked {
		return <-errCh
	}

	// Generate commit message
	message, err := commitFunc(ctx)
	if err != nil {
		p.SetError(err.Error())
		return <-errCh
	}
	p.SetCommitGenerated(message)

	return <-errCh
}
