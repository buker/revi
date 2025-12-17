// Package ai provides a client for interacting with Claude via the Claude Code SDK.
// It handles mode detection, code review execution, and commit message generation
// using the claude-code-sdk-go to communicate with Claude models through the CLI.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	claudecode "github.com/rokrokss/claude-code-sdk-go"

	"github.com/buker/revi/internal/review"
)

// debugEnabled checks if DEBUG environment variable is set
var debugEnabled = os.Getenv("DEBUG") != ""

// debugLog prints a debug message if DEBUG is set
func debugLog(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[AI DEBUG] "+format+"\n", args...)
	}
}

// MaxDiffSize is the maximum size of a diff that can be sent to Claude.
// This is set conservatively to avoid context limits (~100K chars is approximately 25K tokens).
const MaxDiffSize = 100000

// ClientWrapper stores configuration for Claude Code SDK client interactions.
// The actual SDK client is provided via WithClient() pattern for lifecycle management.
type ClientWrapper struct {
	model          string
	streamCallback StreamCallback
}

// NewClientWrapper creates a new ClientWrapper with the specified model.
// Authentication is handled by the Claude Code CLI - users must run 'claude login' first.
// Returns a wrapper that stores configuration; actual client is created via WithClient().
func NewClientWrapper(model string) *ClientWrapper {
	return &ClientWrapper{
		model: model,
	}
}

// SetStreamCallback sets a callback function for receiving streaming content updates.
func (c *ClientWrapper) SetStreamCallback(callback StreamCallback) {
	c.streamCallback = callback
}

// Model returns the configured model name.
func (c *ClientWrapper) Model() string {
	return c.model
}

// StreamCallback returns the configured stream callback.
func (c *ClientWrapper) StreamCallback() StreamCallback {
	return c.streamCallback
}

// RunWithClient executes the provided function with a connected Claude Code SDK client.
// This wraps claudecode.WithClient() and passes the model configuration.
// The client connection is automatically managed - connected before fn runs, disconnected after.
func (c *ClientWrapper) RunWithClient(ctx context.Context, fn func(client claudecode.Client) error) error {
	opts := []claudecode.Option{
		claudecode.WithModel(c.model),
	}

	return claudecode.WithClient(ctx, fn, opts...)
}

// DetectModes asks Claude to analyze the diff and detect relevant review modes.
// Requires a connected SDK client - use within RunWithClient callback.
func (c *ClientWrapper) DetectModes(ctx context.Context, client claudecode.Client, diff string) (*review.DetectionResult, error) {
	diff = truncateDiff(diff)

	prompt := fmt.Sprintf(`Analyze the following git diff and determine which review modes are relevant.

Available modes:
- security: SQL injection, command injection, XSS, authentication issues, secrets exposure, input validation
- performance: N+1 queries, unnecessary loops, memory allocations, blocking calls, caching opportunities
- style: Naming conventions, code patterns, consistency, idiomatic usage, readability
- errors: Missing error checks, swallowed exceptions, unhelpful error messages, edge cases
- testing: Untested code paths, missing assertions, test quality, coverage gaps
- docs: Missing comments, unclear names, outdated comments, API documentation

Respond with ONLY valid JSON in this exact format:
{"modes": ["mode1", "mode2"], "reasoning": "brief explanation"}

Git diff:
%s`, diff)

	var response string
	err := executeWithRetry(ctx, func() error {
		var callErr error
		response, callErr = c.callAPIWithStreaming(ctx, client, prompt, review.Mode(""))
		return callErr
	}, c.streamCallback)

	if err != nil {
		return nil, fmt.Errorf("failed to detect modes: %w", err)
	}

	// Strip markdown code fences if present
	response = stripMarkdownCodeFences(response)

	var result review.DetectionResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse detection result: %w (response: %s)", err, response)
	}

	return &result, nil
}

// RunReview runs a specific review mode on the diff.
// Requires a connected SDK client - use within RunWithClient callback.
func (c *ClientWrapper) RunReview(ctx context.Context, client claudecode.Client, mode review.Mode, diff string) (*review.Result, error) {
	diff = truncateDiff(diff)
	modeInfo := review.GetModeInfo(mode)

	prompt := fmt.Sprintf(`You are a code reviewer focused ONLY on %s concerns.

Focus areas: %s

Review the following git diff and respond with ONLY valid JSON in this exact format:
{
  "mode": "%s",
  "status": "issues_found" or "no_issues",
  "summary": "brief 1-2 sentence summary",
  "issues": [
    {
      "severity": "high|medium|low",
      "description": "issue description",
      "location": "file:line if known",
      "fix": {
        "available": true or false,
        "code": "replacement code with proper indentation (only if available=true)",
        "file_path": "path/to/file.go (only if available=true)",
        "start_line": 42,
        "end_line": 42,
        "explanation": "why this fix works (only if available=true)",
        "reason": "why fix unavailable (only if available=false)",
        "alternatives": ["manual step 1", "manual step 2"]
      }
    }
  ],
  "suggestions": ["suggestion 1", "suggestion 2"]
}

Important:
- Only report issues related to %s
- Be concise and actionable
- If no issues found, return empty issues array and status "no_issues"
- EVERY issue MUST have a concrete fix with available=true. Do NOT report issues you cannot fix.
- For each issue, include a "fix" object:
  - The fix MUST be real, working code - NEVER use TODO comments, placeholder text, or "implement this" stubs
  - Set available=true and provide the complete corrected code in the "code" field
  - The code field must contain the exact replacement text with proper indentation
  - Include file_path, start_line, end_line, and explanation for all fixes
  - Only set available=false in rare cases where the fix truly requires human judgment (e.g., business logic decisions, choosing between multiple valid architectures). In these cases, explain clearly in "reason" why you cannot decide.
  - If you cannot provide a real fix for an issue, do NOT report that issue at all
- Do NOT include fixes that say "add validation here" or "handle error" - show the actual code

Git diff:
%s`, modeInfo.Name, modeInfo.Description, mode, modeInfo.Name, diff)

	var response string
	err := executeWithRetry(ctx, func() error {
		var callErr error
		response, callErr = c.callAPIWithStreaming(ctx, client, prompt, mode)
		return callErr
	}, c.streamCallback)

	if err != nil {
		return &review.Result{
			Mode:   mode,
			Status: review.StatusFailed,
			Error:  err.Error(),
		}, nil
	}

	// Strip markdown code fences if present
	response = stripMarkdownCodeFences(response)

	var result review.Result
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse review result: %w (response: %s)", err, response)
	}

	result.Mode = mode
	if len(result.Issues) > 0 {
		result.Status = review.StatusIssues
	} else {
		result.Status = review.StatusNoIssues
	}

	return &result, nil
}

// CommitMessage represents a generated commit message.
type CommitMessage struct {
	Type    string `json:"type"`
	Scope   string `json:"scope,omitempty"`
	Subject string `json:"subject"`
	Body    string `json:"body,omitempty"`
}

// String returns the formatted commit message in conventional commit format.
func (m *CommitMessage) String() string {
	var msg string
	if m.Scope != "" {
		msg = fmt.Sprintf("%s(%s): %s", m.Type, m.Scope, m.Subject)
	} else {
		msg = fmt.Sprintf("%s: %s", m.Type, m.Subject)
	}
	if m.Body != "" {
		msg += "\n\n" + m.Body
	}
	return msg
}

// GenerateCommitMessage generates a conventional commit message for the diff.
// If context is provided, it will be included in the prompt to explain
// the reasoning behind the change.
// Requires a connected SDK client - use within RunWithClient callback.
func (c *ClientWrapper) GenerateCommitMessage(ctx context.Context, client claudecode.Client, diff string, commitContext string) (*CommitMessage, error) {
	debugLog("GenerateCommitMessage called (diff length: %d, context: %q)", len(diff), commitContext)
	diff = truncateDiff(diff)
	debugLog("Diff after truncation: %d bytes", len(diff))

	contextSection := ""
	if commitContext != "" {
		contextSection = fmt.Sprintf(`
Context (why this change was made):
%s

`, commitContext)
	}

	prompt := fmt.Sprintf(`Generate a conventional commit message for the following git diff.
%s
Respond with ONLY valid JSON in this exact format:
{
  "type": "feat|fix|docs|style|refactor|perf|test|chore",
  "scope": "optional scope",
  "subject": "imperative mood, lowercase, no period, max 50 chars",
  "body": "optional longer description explaining WHY this change was made"
}

Commit types:
- feat: new feature
- fix: bug fix
- docs: documentation only
- style: formatting, no code change
- refactor: code change that neither fixes bug nor adds feature
- perf: performance improvement
- test: adding or fixing tests
- chore: maintenance tasks

Git diff:
%s`, contextSection, diff)

	debugLog("Prompt prepared (length: %d bytes)", len(prompt))

	var response string
	debugLog("Calling executeWithRetry...")
	err := executeWithRetry(ctx, func() error {
		debugLog("Inside retry function, calling callAPIWithStreaming...")
		var callErr error
		response, callErr = c.callAPIWithStreaming(ctx, client, prompt, review.Mode(""))
		debugLog("callAPIWithStreaming returned: err=%v, response length=%d", callErr, len(response))
		return callErr
	}, c.streamCallback)

	if err != nil {
		debugLog("executeWithRetry failed: %v", err)
		return nil, fmt.Errorf("failed to generate commit message: %w", err)
	}

	debugLog("Response received: %s", response)

	// Strip markdown code fences if present
	response = stripMarkdownCodeFences(response)
	debugLog("Response after stripping markdown: %s", response)

	var msg CommitMessage
	if err := json.Unmarshal([]byte(response), &msg); err != nil {
		debugLog("JSON unmarshal failed: %v", err)
		return nil, fmt.Errorf("failed to parse commit message: %w (response: %s)", err, response)
	}

	debugLog("Commit message parsed successfully: type=%s, subject=%s", msg.Type, msg.Subject)
	return &msg, nil
}

// callAPIWithStreaming makes a streaming request via the Claude Code SDK.
// It sends progressive content updates via the streamCallback and returns the complete response.
func (c *ClientWrapper) callAPIWithStreaming(ctx context.Context, client claudecode.Client, prompt string, mode review.Mode) (string, error) {
	debugLog("callAPIWithStreaming: starting (prompt length: %d, mode: %s)", len(prompt), mode)

	// Send query to Claude
	debugLog("callAPIWithStreaming: sending query to Claude...")
	if err := client.Query(ctx, prompt); err != nil {
		debugLog("callAPIWithStreaming: Query failed: %v", err)
		return "", fmt.Errorf("failed to send query: %w", err)
	}
	debugLog("callAPIWithStreaming: query sent successfully")

	var contentBuilder strings.Builder

	// Receive and process messages from the response channel
	debugLog("callAPIWithStreaming: starting to receive messages...")
	msgCount := 0
	for msg := range client.ReceiveMessages(ctx) {
		msgCount++
		debugLog("callAPIWithStreaming: received message #%d (type: %T)", msgCount, msg)

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			debugLog("callAPIWithStreaming: processing AssistantMessage with %d content blocks", len(m.Content))
			// Process content blocks in assistant messages
			for i, block := range m.Content {
				debugLog("callAPIWithStreaming: processing block #%d (type: %T)", i, block)
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					debugLog("callAPIWithStreaming: TextBlock content length: %d", len(textBlock.Text))
					contentBuilder.WriteString(textBlock.Text)
					sendStreamContent(c.streamCallback, mode, textBlock.Text)
				}
			}
		case *claudecode.ResultMessage:
			debugLog("callAPIWithStreaming: received ResultMessage (IsError: %v)", m.IsError)
			// Result message indicates completion - return regardless of error status
			if m.IsError {
				debugLog("callAPIWithStreaming: error result, returning error")
				if contentBuilder.Len() > 0 {
					sendStreamContent(c.streamCallback, mode, "...")
				}
				return "", fmt.Errorf("API error in result message")
			}
			// Success case: operation completed successfully, return the collected content
			result := contentBuilder.String()
			debugLog("callAPIWithStreaming: success result, returning content (length: %d)", len(result))
			return result, nil
		default:
			debugLog("callAPIWithStreaming: unknown message type: %T", msg)
		}
	}

	debugLog("callAPIWithStreaming: channel closed, returning collected content (length: %d)", contentBuilder.Len())
	return contentBuilder.String(), nil
}

// truncateDiff truncates a diff to MaxDiffSize if it exceeds the limit.
// It attempts to truncate at a line boundary for cleaner output.
func truncateDiff(diff string) string {
	if len(diff) <= MaxDiffSize {
		return diff
	}

	// Find a good truncation point (end of a line) within the last 1000 chars
	truncateAt := MaxDiffSize
	for i := MaxDiffSize; i > MaxDiffSize-1000 && i > 0; i-- {
		if diff[i] == '\n' {
			truncateAt = i
			break
		}
	}

	return diff[:truncateAt] + "\n\n[... diff truncated due to size limits ...]"
}

// stripMarkdownCodeFences removes markdown code fence wrappers from AI responses.
// Claude sometimes wraps JSON responses in ```json ... ``` or ``` ... ``` blocks.
// This function extracts the content between the fences, or returns the input unchanged
// if no fences are present.
func stripMarkdownCodeFences(response string) string {
	// Trim leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Check if response starts with code fence
	if !strings.HasPrefix(response, "```") {
		return response
	}

	// Find the end of the opening fence (first newline after ```)
	firstNewline := strings.Index(response, "\n")
	if firstNewline == -1 {
		return response // Malformed, return as-is
	}

	// Find the closing fence (``` at start of a line near the end)
	lastFence := strings.LastIndex(response, "\n```")
	if lastFence == -1 {
		// Try without leading newline (in case it's at the very end)
		if strings.HasSuffix(response, "```") {
			lastFence = len(response) - 3
		} else {
			return response // Malformed, return as-is
		}
	}

	// Extract content between fences
	content := response[firstNewline+1 : lastFence]
	return strings.TrimSpace(content)
}

// Client is a backward-compatible wrapper for the old API.
// Deprecated: Use ClientWrapper with RunWithClient() pattern instead.
// This type will be removed in a future version.
type Client = ClientWrapper

// NewClient creates a new AI client wrapper.
// Authentication is handled by the Claude Code CLI - users must run 'claude login' first.
func NewClient(model string) (*Client, error) {
	return NewClientWrapper(model), nil
}
