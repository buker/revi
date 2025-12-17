package ai

import (
	"context"
	"sync"
	"testing"

	claudecode "github.com/rokrokss/claude-code-sdk-go"

	"github.com/buker/revi/internal/review"
)

// Task 4.1: Focused tests for API methods with new SDK client

// TestDetectModes_WithSDKClient verifies DetectModes() works correctly with
// the Claude Code SDK client and properly parses JSON responses.
func TestDetectModes_WithSDKClient(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Mock response: valid JSON detection result
	jsonResponse := `{"modes": ["security", "style"], "reasoning": "Found potential security and style issues"}`
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: jsonResponse},
		},
	}
	close(transport.msgChan)

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	// Track streaming callbacks
	var mu sync.Mutex
	var streamedContent []string
	wrapper.SetStreamCallback(func(content StreamContent) {
		mu.Lock()
		defer mu.Unlock()
		streamedContent = append(streamedContent, content.Content)
	})

	var result *review.DetectionResult
	var detectErr error
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		result, detectErr = wrapper.DetectModes(ctx, client, "diff content here")
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if detectErr != nil {
		t.Fatalf("DetectModes() error = %v, want nil", detectErr)
	}

	// Verify result parsing
	if result == nil {
		t.Fatal("DetectModes() result = nil, want non-nil")
	}
	if len(result.Modes) != 2 {
		t.Errorf("DetectModes() modes count = %d, want 2", len(result.Modes))
	}
	if result.Modes[0] != review.ModeSecurity {
		t.Errorf("DetectModes() modes[0] = %v, want %v", result.Modes[0], review.ModeSecurity)
	}
	if result.Modes[1] != review.ModeStyle {
		t.Errorf("DetectModes() modes[1] = %v, want %v", result.Modes[1], review.ModeStyle)
	}
	if result.Reasoning != "Found potential security and style issues" {
		t.Errorf("DetectModes() reasoning = %q, want %q", result.Reasoning, "Found potential security and style issues")
	}

	// Verify streaming callback was invoked
	mu.Lock()
	defer mu.Unlock()
	if len(streamedContent) == 0 {
		t.Error("streaming callback not invoked during DetectModes()")
	}
}

// TestRunReview_WithSDKClient verifies RunReview() works correctly with
// the Claude Code SDK client and properly parses review results with issues.
func TestRunReview_WithSDKClient(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Mock response: valid JSON review result with issues
	jsonResponse := `{
		"mode": "security",
		"status": "issues_found",
		"summary": "Found SQL injection vulnerability",
		"issues": [
			{
				"severity": "high",
				"description": "SQL injection in query builder",
				"location": "db/query.go:42",
				"fix": {
					"available": true,
					"code": "db.Query(stmt, args...)",
					"file_path": "db/query.go",
					"start_line": 42,
					"end_line": 42,
					"explanation": "Use parameterized queries"
				}
			}
		],
		"suggestions": ["Consider using an ORM"]
	}`
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: jsonResponse},
		},
	}
	close(transport.msgChan)

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	// Track streaming callbacks for the specific mode
	var mu sync.Mutex
	var streamedModes []review.Mode
	wrapper.SetStreamCallback(func(content StreamContent) {
		mu.Lock()
		defer mu.Unlock()
		streamedModes = append(streamedModes, content.Mode)
	})

	var result *review.Result
	var reviewErr error
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		result, reviewErr = wrapper.RunReview(ctx, client, review.ModeSecurity, "diff content here")
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if reviewErr != nil {
		t.Fatalf("RunReview() error = %v, want nil", reviewErr)
	}

	// Verify result parsing
	if result == nil {
		t.Fatal("RunReview() result = nil, want non-nil")
	}
	if result.Mode != review.ModeSecurity {
		t.Errorf("RunReview() mode = %v, want %v", result.Mode, review.ModeSecurity)
	}
	if result.Status != review.StatusIssues {
		t.Errorf("RunReview() status = %v, want %v", result.Status, review.StatusIssues)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("RunReview() issues count = %d, want 1", len(result.Issues))
	}
	if result.Issues[0].Severity != "high" {
		t.Errorf("RunReview() issue severity = %q, want %q", result.Issues[0].Severity, "high")
	}
	if result.Issues[0].Fix == nil || !result.Issues[0].Fix.Available {
		t.Error("RunReview() issue should have available fix")
	}

	// Verify streaming callback was invoked with correct mode
	mu.Lock()
	defer mu.Unlock()
	if len(streamedModes) == 0 {
		t.Error("streaming callback not invoked during RunReview()")
	}
	for _, mode := range streamedModes {
		if mode != review.ModeSecurity {
			t.Errorf("streaming callback mode = %v, want %v", mode, review.ModeSecurity)
		}
	}
}

// TestRunReview_NoIssues verifies RunReview() correctly handles responses
// with no issues found.
func TestRunReview_NoIssues(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Mock response: no issues found
	jsonResponse := `{
		"mode": "performance",
		"status": "no_issues",
		"summary": "No performance issues found",
		"issues": [],
		"suggestions": []
	}`
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: jsonResponse},
		},
	}
	close(transport.msgChan)

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	var result *review.Result
	var reviewErr error
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		result, reviewErr = wrapper.RunReview(ctx, client, review.ModePerformance, "diff content here")
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if reviewErr != nil {
		t.Fatalf("RunReview() error = %v, want nil", reviewErr)
	}

	// Verify no issues status
	if result.Status != review.StatusNoIssues {
		t.Errorf("RunReview() status = %v, want %v", result.Status, review.StatusNoIssues)
	}
	if len(result.Issues) != 0 {
		t.Errorf("RunReview() issues count = %d, want 0", len(result.Issues))
	}
}

// TestGenerateCommitMessage_WithSDKClient verifies GenerateCommitMessage() works
// correctly with the Claude Code SDK client.
func TestGenerateCommitMessage_WithSDKClient(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Mock response: valid JSON commit message
	jsonResponse := `{
		"type": "feat",
		"scope": "api",
		"subject": "add user authentication endpoint",
		"body": "Implements JWT-based authentication for the API"
	}`
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: jsonResponse},
		},
	}
	close(transport.msgChan)

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	var result *CommitMessage
	var commitErr error
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		result, commitErr = wrapper.GenerateCommitMessage(ctx, client, "diff content here", "")
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if commitErr != nil {
		t.Fatalf("GenerateCommitMessage() error = %v, want nil", commitErr)
	}

	// Verify result parsing
	if result == nil {
		t.Fatal("GenerateCommitMessage() result = nil, want non-nil")
	}
	if result.Type != "feat" {
		t.Errorf("GenerateCommitMessage() type = %q, want %q", result.Type, "feat")
	}
	if result.Scope != "api" {
		t.Errorf("GenerateCommitMessage() scope = %q, want %q", result.Scope, "api")
	}
	if result.Subject != "add user authentication endpoint" {
		t.Errorf("GenerateCommitMessage() subject = %q, want %q", result.Subject, "add user authentication endpoint")
	}

	// Verify String() format
	expectedString := "feat(api): add user authentication endpoint\n\nImplements JWT-based authentication for the API"
	if result.String() != expectedString {
		t.Errorf("CommitMessage.String() = %q, want %q", result.String(), expectedString)
	}
}

// TestGenerateCommitMessage_WithContext verifies GenerateCommitMessage() includes
// user-provided context in the prompt and handles responses correctly.
func TestGenerateCommitMessage_WithContext(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Mock response
	jsonResponse := `{
		"type": "fix",
		"scope": "",
		"subject": "resolve authentication timeout issue",
		"body": "Increases timeout from 5s to 30s per user request"
	}`
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: jsonResponse},
		},
	}
	close(transport.msgChan)

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	var result *CommitMessage
	var commitErr error
	commitContext := "User reported auth timeouts in production"
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		result, commitErr = wrapper.GenerateCommitMessage(ctx, client, "diff content here", commitContext)
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if commitErr != nil {
		t.Fatalf("GenerateCommitMessage() error = %v, want nil", commitErr)
	}

	// Verify result
	if result.Type != "fix" {
		t.Errorf("GenerateCommitMessage() type = %q, want %q", result.Type, "fix")
	}

	// Verify String() format without scope
	expectedString := "fix: resolve authentication timeout issue\n\nIncreases timeout from 5s to 30s per user request"
	if result.String() != expectedString {
		t.Errorf("CommitMessage.String() = %q, want %q", result.String(), expectedString)
	}
}

// TestTruncateDiff_SDKAgnostic verifies that truncateDiff() continues to work
// correctly as an SDK-agnostic utility function.
func TestTruncateDiff_SDKAgnostic(t *testing.T) {
	// Small diff - should be unchanged
	smallDiff := "diff --git a/file.go b/file.go\n+added line"
	result := truncateDiff(smallDiff)
	if result != smallDiff {
		t.Errorf("truncateDiff() modified small diff unexpectedly")
	}

	// Exactly at MaxDiffSize - should be unchanged
	atLimit := makeStringOfLength(MaxDiffSize)
	result = truncateDiff(atLimit)
	if result != atLimit {
		t.Errorf("truncateDiff() modified diff at exact limit")
	}

	// Over MaxDiffSize - should be truncated with marker
	overLimit := makeStringOfLength(MaxDiffSize + 5000)
	result = truncateDiff(overLimit)
	if len(result) > MaxDiffSize+100 { // Allow for truncation marker
		t.Errorf("truncateDiff() result too long: %d bytes", len(result))
	}
	expectedSuffix := "[... diff truncated due to size limits ...]"
	if len(result) < len(expectedSuffix) {
		t.Fatal("truncateDiff() result too short")
	}
	if result[len(result)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("truncateDiff() should end with truncation marker")
	}
}
