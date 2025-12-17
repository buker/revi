package ai

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	claudecode "github.com/rokrokss/claude-code-sdk-go"
)

// Retry configuration constants
const (
	maxRateLimitRetries = 3
	maxNetworkRetries   = 1
	maxProcessRetries   = 1
	initialBackoff      = 1 * time.Second
	networkRetryDelay   = 2 * time.Second
	processRetryDelay   = 2 * time.Second
)

// Error messages for user-friendly output
const (
	errMsgCLINotFound = "Claude Code CLI not found. Install with: npm install -g @anthropic-ai/claude-code"
	errMsgAuth        = "Claude CLI authentication required. Run 'claude login' to authenticate."
	errMsgRateLimit   = "rate limit exceeded after 3 retries"
	errMsgNetwork     = "network error: %s"
	errMsgConnection  = "connection to Claude Code CLI failed: %s"
	errMsgProcess     = "Claude Code CLI subprocess failed: %s"
	errMsgServer      = "Claude API error occurred. Please try again later."
	errMsgTimeout     = "request timed out"
)

// StreamCallback is a function that receives streaming content updates
type StreamCallback func(content StreamContent)

// executeWithRetry wraps an API call with retry logic based on error type.
// It handles CLI errors, subprocess failures, network errors, and timeouts
// according to the claude-code-sdk-go error types.
func executeWithRetry(ctx context.Context, fn func() error, streamCallback StreamCallback) error {
	var lastErr error
	rateLimitRetries := 0
	networkRetries := 0
	processRetries := 0
	backoff := initialBackoff

	for {
		// Check context before attempting
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Classify and handle the error
		errType := classifyError(lastErr)

		switch errType {
		case errTypeCLINotFound:
			// CLI not found - no retry, guide user to install
			return errors.New(errMsgCLINotFound)

		case errTypeAuth:
			// Authentication required - no retry, guide user to login
			return errors.New(errMsgAuth)

		case errTypeRateLimit:
			// Rate limit - retry with exponential backoff
			rateLimitRetries++
			if rateLimitRetries > maxRateLimitRetries {
				return errors.New(errMsgRateLimit)
			}
			if err := sleepWithContext(ctx, backoff); err != nil {
				return err
			}
			backoff *= 2 // Exponential backoff

		case errTypeConnection:
			// Connection error - retry once
			networkRetries++
			if networkRetries > maxNetworkRetries {
				return fmt.Errorf(errMsgConnection, extractErrorMsg(lastErr))
			}
			if err := sleepWithContext(ctx, networkRetryDelay); err != nil {
				return err
			}

		case errTypeProcess:
			// Subprocess crash - retry once
			processRetries++
			if processRetries > maxProcessRetries {
				return fmt.Errorf(errMsgProcess, extractProcessErrorMsg(lastErr))
			}
			if err := sleepWithContext(ctx, processRetryDelay); err != nil {
				return err
			}

		case errTypeNetwork:
			// Network error - retry once
			networkRetries++
			if networkRetries > maxNetworkRetries {
				return fmt.Errorf(errMsgNetwork, extractNetworkErrorMsg(lastErr))
			}
			if err := sleepWithContext(ctx, networkRetryDelay); err != nil {
				return err
			}

		case errTypeServer:
			// Server error - no retry
			return errors.New(errMsgServer)

		case errTypeTimeout:
			// Timeout - no retry
			return errors.New(errMsgTimeout)

		default:
			return lastErr
		}
	}
}

// errorType represents the category of error
type errorType int

const (
	errTypeUnknown errorType = iota
	errTypeCLINotFound
	errTypeAuth
	errTypeRateLimit
	errTypeConnection
	errTypeProcess
	errTypeNetwork
	errTypeServer
	errTypeTimeout
)

// classifyError determines the type of error based on SDK error types.
// The claude-code-sdk-go uses specific error types instead of HTTP status codes.
func classifyError(err error) errorType {
	if err == nil {
		return errTypeUnknown
	}

	// Check for context timeout/deadline
	if errors.Is(err, context.DeadlineExceeded) {
		return errTypeTimeout
	}
	if errors.Is(err, context.Canceled) {
		return errTypeUnknown // Let caller handle canceled context
	}

	// Check for Claude Code SDK error types
	var cliNotFoundErr *claudecode.CLINotFoundError
	if errors.As(err, &cliNotFoundErr) {
		return errTypeCLINotFound
	}

	var processErr *claudecode.ProcessError
	if errors.As(err, &processErr) {
		return errTypeProcess
	}

	var connectionErr *claudecode.ConnectionError
	if errors.As(err, &connectionErr) {
		return errTypeConnection
	}

	var jsonDecodeErr *claudecode.JSONDecodeError
	if errors.As(err, &jsonDecodeErr) {
		// JSON decode errors are typically not retryable
		return errTypeUnknown
	}

	var messageParseErr *claudecode.MessageParseError
	if errors.As(err, &messageParseErr) {
		// Message parse errors are typically not retryable
		return errTypeUnknown
	}

	// Check for network errors (for backward compatibility and wrapped errors)
	if isNetworkError(err) {
		return errTypeNetwork
	}

	return errTypeUnknown
}

// isNetworkError checks if an error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common network error types
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	return false
}

// extractErrorMsg extracts a user-friendly message from an error
func extractErrorMsg(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}

// extractProcessErrorMsg extracts details from a ProcessError
func extractProcessErrorMsg(err error) string {
	var processErr *claudecode.ProcessError
	if errors.As(err, &processErr) {
		if processErr.Stderr != "" {
			return processErr.Stderr
		}
		if processErr.ExitCode != 0 {
			return fmt.Sprintf("exit code %d", processErr.ExitCode)
		}
	}
	return err.Error()
}

// extractNetworkErrorMsg extracts a user-friendly message from a network error
func extractNetworkErrorMsg(err error) string {
	if err == nil {
		return "unknown network error"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.Err
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Op + " failed"
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "connection timed out"
		}
		return "connection failed"
	}

	return err.Error()
}

// sleepWithContext sleeps for the specified duration, respecting context cancellation
func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
