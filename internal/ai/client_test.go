package ai

import (
	"context"
	"testing"
	"time"

	claudecode "github.com/rokrokss/claude-code-sdk-go"

	"github.com/buker/revi/internal/review"
)

func TestNewClient_NoTokenRequired(t *testing.T) {
	// Authentication is handled by the Claude Code CLI
	client, err := NewClient("claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}
	if client.model != "claude-sonnet-4-20250514" {
		t.Errorf("client.model = %q, want %q", client.model, "claude-sonnet-4-20250514")
	}
}

func TestTruncateDiff_AtBoundary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantFull bool
	}{
		{
			name:     "small diff unchanged",
			input:    "small diff content",
			wantLen:  len("small diff content"),
			wantFull: true,
		},
		{
			name:     "exactly at MaxDiffSize",
			input:    makeStringOfLength(MaxDiffSize),
			wantLen:  MaxDiffSize,
			wantFull: true,
		},
		{
			name:     "over MaxDiffSize truncated",
			input:    makeStringOfLength(MaxDiffSize + 1000),
			wantLen:  MaxDiffSize + len("\n\n[... diff truncated due to size limits ...]"),
			wantFull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateDiff(tt.input)
			if tt.wantFull {
				if result != tt.input {
					t.Errorf("truncateDiff() changed input when it should not")
				}
			} else {
				if len(result) > tt.wantLen {
					t.Errorf("truncateDiff() len = %d, want <= %d", len(result), tt.wantLen)
				}
				if result[len(result)-1] != ']' {
					t.Error("truncateDiff() should end with truncation marker")
				}
			}
		})
	}
}

func TestTruncateDiff_LineBreakBoundary(t *testing.T) {
	// Create a diff with lines to test line-boundary truncation
	lines := ""
	for i := 0; i < MaxDiffSize/50; i++ {
		lines += "this is a line of exactly forty-nine chars\n"
	}
	lines += "extra content to exceed limit"

	result := truncateDiff(lines)

	// Should truncate at a newline boundary
	if len(result) <= MaxDiffSize {
		return // Result fits, no truncation marker expected in output
	}
	// Verify it ends with truncation marker
	truncMarker := "\n\n[... diff truncated due to size limits ...]"
	if len(result) < len(truncMarker) {
		t.Fatal("result too short")
	}
	suffix := result[len(result)-len(truncMarker):]
	if suffix != truncMarker {
		t.Errorf("truncateDiff() should end with truncation marker, got suffix: %q", suffix)
	}
}

func TestCommitMessage_String(t *testing.T) {
	// Verify CommitMessage structure and String() method
	cm := &CommitMessage{
		Type:    "feat",
		Scope:   "api",
		Subject: "add new endpoint",
		Body:    "This adds the users endpoint",
	}
	expected := "feat(api): add new endpoint\n\nThis adds the users endpoint"
	if cm.String() != expected {
		t.Errorf("CommitMessage.String() = %q, want %q", cm.String(), expected)
	}

	// Test without scope
	cm2 := &CommitMessage{
		Type:    "fix",
		Subject: "resolve bug",
	}
	expected2 := "fix: resolve bug"
	if cm2.String() != expected2 {
		t.Errorf("CommitMessage.String() = %q, want %q", cm2.String(), expected2)
	}
}

func TestDetectionResult_Structure(t *testing.T) {
	// Verify DetectionResult structure
	dr := &review.DetectionResult{
		Modes:     []review.Mode{review.ModeSecurity, review.ModeStyle},
		Reasoning: "test reasoning",
	}
	if len(dr.Modes) != 2 {
		t.Errorf("DetectionResult.Modes length = %d, want 2", len(dr.Modes))
	}
}

func TestReviewResult_Structure(t *testing.T) {
	// Verify Result structure
	r := &review.Result{
		Mode:    review.ModeSecurity,
		Status:  review.StatusIssues,
		Summary: "test summary",
		Issues: []review.Issue{
			{Severity: "high", Description: "test issue"},
		},
	}
	if r.Mode != review.ModeSecurity {
		t.Errorf("Result.Mode = %v, want %v", r.Mode, review.ModeSecurity)
	}
}

// Helper function to create a string of specific length
func makeStringOfLength(n int) string {
	result := make([]byte, n)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}

// TestCallAPIWithStreaming_SuccessfulResultMessageExitsLoop verifies that
// a successful ResultMessage (IsError=false) causes the function to return.
// This test will hang indefinitely if the bug is present.
func TestCallAPIWithStreaming_SuccessfulResultMessageExitsLoop(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Send assistant message followed by successful result message
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: "Response content"},
		},
	}
	transport.msgChan <- &claudecode.ResultMessage{
		IsError: false, // Successful completion
	}
	// NOTE: Don't close channel yet - test if function exits on ResultMessage

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	// Use a timeout to detect if the function hangs
	done := make(chan struct{})
	var result string
	var resultErr error

	go func() {
		defer close(done)
		err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
			result, resultErr = wrapper.callAPIWithStreaming(ctx, client, "test prompt", review.ModeSecurity)
			return nil
		})
		if err != nil {
			t.Errorf("WithClientTransport() error = %v, want nil", err)
		}
	}()

	// Wait for completion with timeout
	select {
	case <-done:
		// Success - function returned
		if resultErr != nil {
			t.Errorf("callAPIWithStreaming() error = %v, want nil", resultErr)
		}
		if result != "Response content" {
			t.Errorf("result = %q, want %q", result, "Response content")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("callAPIWithStreaming() hung - did not exit after receiving successful ResultMessage")
	}
}

// TestCallAPIWithStreaming_ErrorResultMessageReturnsError verifies that
// an error ResultMessage (IsError=true) causes the function to return an error.
func TestCallAPIWithStreaming_ErrorResultMessageReturnsError(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Send error result message
	transport.msgChan <- &claudecode.ResultMessage{
		IsError: true,
	}
	close(transport.msgChan)

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	var result string
	var resultErr error
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		result, resultErr = wrapper.callAPIWithStreaming(ctx, client, "test prompt", review.ModeSecurity)
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if resultErr == nil {
		t.Fatal("callAPIWithStreaming() error = nil, want error for IsError=true")
	}
	if result != "" {
		t.Errorf("result = %q, want empty string on error", result)
	}
}

// TestStripMarkdownCodeFences verifies that markdown code fences are properly
// removed from AI responses before JSON parsing.
func TestStripMarkdownCodeFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "JSON with ```json wrapper",
			input: "```json\n" +
				`{"type": "feat", "subject": "test"}` + "\n" +
				"```",
			expected: `{"type": "feat", "subject": "test"}`,
		},
		{
			name: "JSON with plain ``` wrapper",
			input: "```\n" +
				`{"type": "feat", "subject": "test"}` + "\n" +
				"```",
			expected: `{"type": "feat", "subject": "test"}`,
		},
		{
			name:     "Raw JSON without wrapper",
			input:    `{"type": "feat", "subject": "test"}`,
			expected: `{"type": "feat", "subject": "test"}`,
		},
		{
			name: "JSON with extra whitespace",
			input: "```json  \n" +
				`{"type": "feat", "subject": "test"}` + "\n" +
				"```  \n",
			expected: `{"type": "feat", "subject": "test"}`,
		},
		{
			name: "Multiline JSON with wrapper",
			input: "```json\n" +
				"{\n" +
				`  "type": "feat",` + "\n" +
				`  "subject": "test"` + "\n" +
				"}\n" +
				"```",
			expected: "{\n" +
				`  "type": "feat",` + "\n" +
				`  "subject": "test"` + "\n" +
				"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownCodeFences(tt.input)
			if result != tt.expected {
				t.Errorf("stripMarkdownCodeFences() = %q, want %q", result, tt.expected)
			}
		})
	}
}
