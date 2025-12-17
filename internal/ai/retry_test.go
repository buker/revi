package ai

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// TestExecuteWithRetry_NetworkError_RetriesOnceAfter2s tests that network errors
// are retried once with a 2 second delay.
func TestExecuteWithRetry_NetworkError_RetriesOnceAfter2s(t *testing.T) {
	callCount := 0
	callTimes := make([]time.Time, 0, 2)
	fn := func() error {
		callTimes = append(callTimes, time.Now())
		callCount++
		return &net.DNSError{Err: "no such host", IsNotFound: true}
	}

	err := executeWithRetry(context.Background(), fn, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should have 2 calls total (1 initial + 1 retry)
	if callCount != 2 {
		t.Errorf("expected 2 calls (1 initial + 1 retry), got %d", callCount)
	}

	// Verify 2s delay
	tolerance := 200 * time.Millisecond
	if len(callTimes) >= 2 {
		delay := callTimes[1].Sub(callTimes[0])
		if delay < 2*time.Second-tolerance || delay > 2*time.Second+tolerance {
			t.Errorf("network retry delay = %v, want ~2s", delay)
		}
	}

	expectedMsg := "network error: no such host"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestExecuteWithRetry_Success tests that successful calls return without error.
func TestExecuteWithRetry_Success(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := executeWithRetry(context.Background(), fn, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

// TestExecuteWithRetry_ContextCanceled tests that canceled contexts are handled properly.
func TestExecuteWithRetry_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fn := func() error {
		return errors.New("should not be called")
	}

	err := executeWithRetry(ctx, fn, nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestExecuteWithRetry_TimeoutError tests that context deadline exceeded is handled.
func TestExecuteWithRetry_TimeoutError(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		return context.DeadlineExceeded
	}

	err := executeWithRetry(context.Background(), fn, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for timeout), got %d", callCount)
	}
	expectedMsg := "request timed out"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestExecuteWithRetry_UnknownError tests that unknown errors are passed through.
func TestExecuteWithRetry_UnknownError(t *testing.T) {
	callCount := 0
	unknownErr := errors.New("unknown error")
	fn := func() error {
		callCount++
		return unknownErr
	}

	err := executeWithRetry(context.Background(), fn, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for unknown errors), got %d", callCount)
	}
	if err.Error() != unknownErr.Error() {
		t.Errorf("error message = %q, want %q", err.Error(), unknownErr.Error())
	}
}
