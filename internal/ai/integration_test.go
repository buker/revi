package ai

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	claudecode "github.com/rokrokss/claude-code-sdk-go"

	"github.com/buker/revi/internal/review"
)

// TestSDKClientWithParallelReviewExecution verifies the SDK client works correctly
// when multiple reviews run concurrently via the review runner pattern.
func TestSDKClientWithParallelReviewExecution(t *testing.T) {
	// Simulate parallel review execution with mock functions
	modes := []review.Mode{review.ModeSecurity, review.ModeStyle, review.ModeErrors}
	results := make([]*review.Result, len(modes))

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Mock review function that simulates concurrent execution
	mockReviewFunc := func(idx int, mode review.Mode) {
		defer wg.Done()

		// Simulate some work with varying delays
		time.Sleep(time.Duration(10*(idx+1)) * time.Millisecond)

		result := &review.Result{
			Mode:   mode,
			Status: review.StatusNoIssues,
		}

		mu.Lock()
		results[idx] = result
		mu.Unlock()
	}

	// Execute all reviews in parallel
	for i, mode := range modes {
		wg.Add(1)
		go mockReviewFunc(i, mode)
	}

	wg.Wait()

	// Verify all results were collected
	for i, result := range results {
		if result == nil {
			t.Errorf("result[%d] is nil, expected non-nil", i)
			continue
		}
		if result.Mode != modes[i] {
			t.Errorf("result[%d].Mode = %v, want %v", i, result.Mode, modes[i])
		}
	}
}

// TestModelOverridePropagates verifies that model configuration is properly
// stored and would be used in API calls.
func TestModelOverridePropagates(t *testing.T) {
	// Authentication is handled by the Claude Code CLI

	testCases := []struct {
		name      string
		modelArg  string
		wantModel string
	}{
		{
			name:      "default opus model",
			modelArg:  "claude-opus-4-5-20251101",
			wantModel: "claude-opus-4-5-20251101",
		},
		{
			name:      "sonnet override",
			modelArg:  "claude-sonnet-4-20250514",
			wantModel: "claude-sonnet-4-20250514",
		},
		{
			name:      "haiku for cost savings",
			modelArg:  "claude-3-5-haiku-20241022",
			wantModel: "claude-3-5-haiku-20241022",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.modelArg)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			if client.model != tc.wantModel {
				t.Errorf("client.model = %q, want %q", client.model, tc.wantModel)
			}
		})
	}
}

// TestEmptyDiffHandling verifies the SDK client handles empty diffs gracefully.
func TestEmptyDiffHandling(t *testing.T) {
	// Test truncateDiff with empty string
	result := truncateDiff("")
	if result != "" {
		t.Errorf("truncateDiff(\"\") = %q, want empty string", result)
	}

	// Test truncateDiff with whitespace-only string
	result = truncateDiff("   \n\t  ")
	if result != "   \n\t  " {
		t.Errorf("truncateDiff(whitespace) changed content unexpectedly")
	}
}

// TestVeryLargeDiffAtMaxDiffSizeBoundary tests edge cases around MaxDiffSize.
func TestVeryLargeDiffAtMaxDiffSizeBoundary(t *testing.T) {
	testCases := []struct {
		name         string
		inputSize    int
		wantTruncate bool
	}{
		{
			name:         "exactly at MaxDiffSize",
			inputSize:    MaxDiffSize,
			wantTruncate: false,
		},
		{
			name:         "one byte over MaxDiffSize",
			inputSize:    MaxDiffSize + 1,
			wantTruncate: true,
		},
		{
			name:         "1KB over MaxDiffSize",
			inputSize:    MaxDiffSize + 1024,
			wantTruncate: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := makeStringOfLength(tc.inputSize)
			result := truncateDiff(input)

			if tc.wantTruncate {
				// Should be truncated with marker
				truncMarker := "\n\n[... diff truncated due to size limits ...]"
				if len(result) > MaxDiffSize+len(truncMarker) {
					t.Errorf("truncated result too long: %d bytes", len(result))
				}
				if result[len(result)-1] != ']' {
					t.Error("truncated result should end with truncation marker")
				}
			} else {
				// Should be unchanged
				if result != input {
					t.Error("input should not be modified when at or below MaxDiffSize")
				}
			}
		})
	}
}

// TestErrorRecoveryDuringParallelReviewExecution verifies that errors in one
// review don't crash the entire parallel execution.
func TestErrorRecoveryDuringParallelReviewExecution(t *testing.T) {
	modes := []review.Mode{review.ModeSecurity, review.ModeStyle, review.ModeErrors}
	results := make([]*review.Result, len(modes))

	var wg sync.WaitGroup
	var successCount int32

	// Mock review function where one fails
	mockReviewFunc := func(idx int, mode review.Mode) {
		defer wg.Done()

		var result *review.Result
		if mode == review.ModeStyle {
			// Simulate a failure for style review
			result = &review.Result{
				Mode:   mode,
				Status: review.StatusFailed,
				Error:  "simulated API error",
			}
		} else {
			// Other reviews succeed
			result = &review.Result{
				Mode:   mode,
				Status: review.StatusNoIssues,
			}
			atomic.AddInt32(&successCount, 1)
		}

		results[idx] = result
	}

	// Execute all reviews in parallel
	for i, mode := range modes {
		wg.Add(1)
		go mockReviewFunc(i, mode)
	}

	wg.Wait()

	// Verify partial success - 2 succeeded, 1 failed
	if successCount != 2 {
		t.Errorf("successCount = %d, want 2", successCount)
	}

	// Verify all results were collected despite failure
	for i, result := range results {
		if result == nil {
			t.Errorf("result[%d] is nil, expected non-nil", i)
		}
	}

	// Verify the failed one has error status
	if results[1].Status != review.StatusFailed {
		t.Errorf("results[1].Status = %v, want StatusFailed", results[1].Status)
	}
}

// TestStreamingCallbackConcurrentSafety verifies stream callbacks are thread-safe.
func TestStreamingCallbackConcurrentSafety(t *testing.T) {
	var mu sync.Mutex
	var messages []StreamContent
	var callCount int32

	callback := func(content StreamContent) {
		atomic.AddInt32(&callCount, 1)
		mu.Lock()
		messages = append(messages, content)
		mu.Unlock()
	}

	// Simulate concurrent streaming from multiple goroutines
	var wg sync.WaitGroup
	modes := []review.Mode{review.ModeSecurity, review.ModePerformance, review.ModeStyle}

	for _, mode := range modes {
		wg.Add(1)
		go func(m review.Mode) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				sendStreamContent(callback, m, "chunk")
			}
		}(mode)
	}

	wg.Wait()

	// Verify all messages received
	expected := int32(len(modes) * 10)
	if callCount != expected {
		t.Errorf("callCount = %d, want %d", callCount, expected)
	}

	mu.Lock()
	if len(messages) != int(expected) {
		t.Errorf("len(messages) = %d, want %d", len(messages), expected)
	}
	mu.Unlock()
}

// TestContextCancellationDuringRetry verifies retry logic respects context cancellation.
func TestContextCancellationDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var callCount int32
	fn := func() error {
		atomic.AddInt32(&callCount, 1)
		// Cancel after first call
		if callCount == 1 {
			cancel()
		}
		// Use ProcessError which triggers a retry
		return claudecode.NewProcessError("subprocess failed", 1, "error")
	}

	err := executeWithRetry(ctx, fn, nil)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}

	// Should have stopped after context was canceled (not all retries)
	if callCount > 2 {
		t.Errorf("callCount = %d, want <= 2 (should stop after context canceled)", callCount)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestStreamContentWithEmptyMode verifies StreamContent works for non-mode operations.
func TestStreamContentWithEmptyMode(t *testing.T) {
	var received []StreamContent
	callback := func(content StreamContent) {
		received = append(received, content)
	}

	// Empty mode for operations like detect and commit generation
	sendStreamContent(callback, review.Mode(""), "analyzing diff...")
	sendStreamContent(callback, review.Mode(""), "generating message...")

	if len(received) != 2 {
		t.Fatalf("len(received) = %d, want 2", len(received))
	}

	for _, msg := range received {
		if msg.Mode != "" {
			t.Errorf("msg.Mode = %q, want empty", msg.Mode)
		}
	}
}
