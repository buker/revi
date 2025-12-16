// Package claude provides a client for interacting with the Claude CLI.
// It handles mode detection, code review execution, and commit message generation
// by invoking the Claude CLI with structured prompts and parsing JSON responses.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/buker/revi/internal/review"
)

// MaxDiffSize is the maximum size of a diff that can be sent to Claude
// This is set conservatively to avoid API limits (~100K chars ≈ 25K tokens)
const MaxDiffSize = 100000

// Client handles communication with Claude CLI
type Client struct {
	path    string
	timeout time.Duration
}

// NewClient creates a new Claude client
func NewClient(path string, timeoutSeconds int) *Client {
	if path == "" {
		path = "claude"
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}
	return &Client{
		path:    path,
		timeout: time.Duration(timeoutSeconds) * time.Second,
	}
}

// DetectModes asks Claude to analyze the diff and detect relevant review modes
func (c *Client) DetectModes(ctx context.Context, diff string) (*review.DetectionResult, error) {
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

	output, err := c.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to detect modes: %w", err)
	}

	var result review.DetectionResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse detection result: %w (output: %s)", err, string(output))
	}

	return &result, nil
}

// RunReview runs a specific review mode on the diff
func (c *Client) RunReview(ctx context.Context, mode review.Mode, diff string) (*review.Result, error) {
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

	output, err := c.run(ctx, prompt)
	if err != nil {
		return &review.Result{
			Mode:   mode,
			Status: review.StatusFailed,
			Error:  err.Error(),
		}, nil
	}

	var result review.Result
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse review result: %w (output: %s)", err, string(output))
	}

	result.Mode = mode
	if len(result.Issues) > 0 {
		result.Status = review.StatusIssues
	} else {
		result.Status = review.StatusNoIssues
	}

	return &result, nil
}

// CommitMessage represents a generated commit message
type CommitMessage struct {
	Type    string `json:"type"`
	Scope   string `json:"scope,omitempty"`
	Subject string `json:"subject"`
	Body    string `json:"body,omitempty"`
}

// String returns the formatted commit message
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
func (c *Client) GenerateCommitMessage(ctx context.Context, diff string, context string) (*CommitMessage, error) {
	diff = truncateDiff(diff)

	contextSection := ""
	if context != "" {
		contextSection = fmt.Sprintf(`
Context (why this change was made):
%s

`, context)
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

	output, err := c.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate commit message: %w", err)
	}

	var msg CommitMessage
	if err := json.Unmarshal(output, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse commit message: %w (output: %s)", err, string(output))
	}

	return &msg, nil
}

// run executes the claude CLI with the given prompt
func (c *Client) run(ctx context.Context, prompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Use stdin for the prompt to handle long content (like git diffs)
	cmd := exec.CommandContext(ctx, c.path, "-p", "--output-format", "json")

	// Pass prompt via stdin
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("claude timed out after %v", c.timeout)
		}
		// Try to parse error from stdout (Claude CLI returns JSON even on errors)
		if stdout.Len() > 0 {
			var errResp struct {
				Type    string `json:"type"`
				IsError bool   `json:"is_error"`
				Result  string `json:"result"`
			}
			if jsonErr := json.Unmarshal(stdout.Bytes(), &errResp); jsonErr == nil && errResp.IsError {
				return nil, fmt.Errorf("claude error: %s", errResp.Result)
			}
		}
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return nil, fmt.Errorf("claude failed: %w (output: %s)", err, errMsg)
	}

	// Parse the Claude CLI wrapper response.
	// The Claude CLI (--output-format json) wraps the AI's response in a JSON envelope
	// with metadata about the request.
	output := stdout.Bytes()

	// Claude CLI response format:
	// {
	//   "type": "result",           // Response type identifier
	//   "is_error": false,          // Whether the request failed
	//   "result": "..."             // The actual AI response (may contain JSON as a string)
	// }
	var wrapper struct {
		Type    string `json:"type"`
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}

	if err := json.Unmarshal(output, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse claude response: %w (output: %s)", err, string(output))
	}

	if wrapper.IsError {
		return nil, fmt.Errorf("claude returned error: %s", wrapper.Result)
	}

	return extractJSONFromClaudeResult(wrapper.Result)
}

// IsAvailable checks if the Claude CLI is available
func (c *Client) IsAvailable() error {
	cmd := exec.Command(c.path, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude CLI not found at '%s': %w\nInstall from: https://claude.ai/download", c.path, err)
	}
	return nil
}

// truncateDiff truncates a diff to MaxDiffSize if it's too large
func truncateDiff(diff string) string {
	if len(diff) <= MaxDiffSize {
		return diff
	}

	// Find a good truncation point (end of a line)
	truncateAt := MaxDiffSize
	for i := MaxDiffSize; i > MaxDiffSize-1000 && i > 0; i-- {
		if diff[i] == '\n' {
			truncateAt = i
			break
		}
	}

	return diff[:truncateAt] + "\n\n[... diff truncated due to size limits ...]"
}

// extractJSONFromClaudeResult extracts the first JSON object from a Claude response string.
//
// Claude often wraps JSON in markdown code blocks or adds extra text, e.g.:
// ```json
// {"key": "value"}
// ```
//
// This function trims markdown fences, finds the first '{', then scans for the
// matching closing brace while correctly ignoring braces inside strings.
func extractJSONFromClaudeResult(result string) ([]byte, error) {
	// Strip markdown code block formatting if present.
	// Handle both ```json and plain ``` markers.
	result = strings.TrimSpace(result)
	if strings.HasPrefix(result, "```json") {
		result = strings.TrimPrefix(result, "```json")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)
	} else if strings.HasPrefix(result, "```") {
		result = strings.TrimPrefix(result, "```")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)
	}

	// Find the start of the JSON object.
	// Claude may include explanatory text before/after the JSON,
	// so we locate the first '{' character.
	start := strings.Index(result, "{")
	if start == -1 {
		return nil, fmt.Errorf("no JSON object found in result: %s", result)
	}

	// Find the matching closing brace by tracking brace depth.
	// This handles nested objects correctly (e.g., {"a": {"b": 1}}).
	// It also correctly ignores braces inside strings (e.g., {"data": "{"}).
	depth := 0
	end := -1
	inString := false
	for i := start; i < len(result); i++ {
		c := result[i]

		// Handle string boundaries (accounting for escaped quotes)
		if c == '"' {
			// Count consecutive backslashes before this quote
			backslashes := 0
			for j := i - 1; j >= 0 && result[j] == '\\'; j-- {
				backslashes++
			}
			// Quote is escaped only if preceded by odd number of backslashes
			if backslashes%2 == 0 {
				inString = !inString
			}
			continue
		}

		// Skip brace counting while inside a string
		if inString {
			continue
		}

		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if end == -1 {
		return nil, fmt.Errorf("incomplete JSON object in result: %s", result)
	}

	// Return only the extracted JSON object bytes.
	return []byte(result[start:end]), nil
}
