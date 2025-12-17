package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	claudecode "github.com/rokrokss/claude-code-sdk-go"
)

// TestClassifyError_CLINotFound tests that CLINotFoundError is classified correctly.
// When the Claude Code CLI is not installed, users need to install it.
func TestClassifyError_CLINotFound(t *testing.T) {
	err := claudecode.NewCLINotFoundError("/usr/local/bin/claude", "Claude CLI not found")
	errType := classifyError(err)
	if errType != errTypeCLINotFound {
		t.Errorf("classifyError(CLINotFoundError) = %v, want errTypeCLINotFound", errType)
	}
}

// TestClassifyError_ProcessError tests that ProcessError is classified as a subprocess error.
// Process errors indicate the CLI subprocess crashed or failed.
func TestClassifyError_ProcessError(t *testing.T) {
	err := claudecode.NewProcessError("subprocess failed", 1, "error output")
	errType := classifyError(err)
	if errType != errTypeProcess {
		t.Errorf("classifyError(ProcessError) = %v, want errTypeProcess", errType)
	}
}

// TestClassifyError_ConnectionError tests that ConnectionError is classified correctly.
// Connection errors may indicate IPC failures or CLI communication issues.
func TestClassifyError_ConnectionError(t *testing.T) {
	err := claudecode.NewConnectionError("connection failed", nil)
	errType := classifyError(err)
	if errType != errTypeConnection {
		t.Errorf("classifyError(ConnectionError) = %v, want errTypeConnection", errType)
	}
}

// TestExecuteWithRetry_CLINotFound tests that CLI not found errors are not retried
// and provide actionable guidance to install the CLI.
func TestExecuteWithRetry_CLINotFound(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		return claudecode.NewCLINotFoundError("", "Claude Code CLI not found")
	}

	err := executeWithRetry(context.Background(), fn, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for CLI not found), got %d", callCount)
	}
	// Verify the error message guides users to install the CLI
	expectedMsg := "Claude Code CLI not found. Install with: npm install -g @anthropic-ai/claude-code"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestExecuteWithRetry_ProcessError_RetriesOnce tests that subprocess failures
// are retried once before giving up.
func TestExecuteWithRetry_ProcessError_RetriesOnce(t *testing.T) {
	callCount := 0
	callTimes := make([]time.Time, 0, 2)
	fn := func() error {
		callTimes = append(callTimes, time.Now())
		callCount++
		return claudecode.NewProcessError("subprocess crashed", 1, "signal: killed")
	}

	err := executeWithRetry(context.Background(), fn, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should have 2 calls total (1 initial + 1 retry for process errors)
	if callCount != 2 {
		t.Errorf("expected 2 calls (1 initial + 1 retry), got %d", callCount)
	}

	// Verify retry delay (2s)
	tolerance := 200 * time.Millisecond
	if len(callTimes) >= 2 {
		delay := callTimes[1].Sub(callTimes[0])
		if delay < 2*time.Second-tolerance || delay > 2*time.Second+tolerance {
			t.Errorf("process retry delay = %v, want ~2s", delay)
		}
	}

	// Verify error message mentions subprocess failure and contains stderr
	errMsg := err.Error()
	if !strings.Contains(errMsg, "subprocess") && !strings.Contains(errMsg, "CLI") {
		t.Errorf("error message should mention subprocess/CLI failure, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "signal: killed") {
		t.Errorf("error message should contain stderr content, got: %q", errMsg)
	}
}
