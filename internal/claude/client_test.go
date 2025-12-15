package claude

import (
	"fmt"
	"strings"
	"testing"
)

// extractJSON is a helper that exposes the JSON extraction logic for testing.
// It mirrors the JSON extraction logic in client.go's run() method to test
// various edge cases in JSON parsing from Claude CLI responses.
func extractJSON(result string) (string, error) {
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
		return "", fmt.Errorf("no JSON object found in result: %s", result)
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
		return "", fmt.Errorf("incomplete JSON object in result: %s", result)
	}

	return result[start:end], nil
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Normal JSON responses
		{
			name:    "simple JSON object",
			input:   `{"key": "value"}`,
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "JSON with whitespace around",
			input:   `  {"key": "value"}  `,
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "JSON with text before",
			input:   `Here is the JSON: {"key": "value"}`,
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "JSON with text after",
			input:   `{"key": "value"} That's the result.`,
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "JSON in markdown code block",
			input:   "```json\n{\"key\": \"value\"}\n```",
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "JSON in plain code block",
			input:   "```\n{\"key\": \"value\"}\n```",
			want:    `{"key": "value"}`,
			wantErr: false,
		},

		// Nested JSON structures
		{
			name:    "nested objects",
			input:   `{"outer": {"inner": "value"}}`,
			want:    `{"outer": {"inner": "value"}}`,
			wantErr: false,
		},
		{
			name:    "deeply nested objects",
			input:   `{"a": {"b": {"c": {"d": "deep"}}}}`,
			want:    `{"a": {"b": {"c": {"d": "deep"}}}}`,
			wantErr: false,
		},
		{
			name:    "multiple nested objects",
			input:   `{"first": {"a": 1}, "second": {"b": 2}}`,
			want:    `{"first": {"a": 1}, "second": {"b": 2}}`,
			wantErr: false,
		},

		// JSON with escaped quotes inside strings
		{
			name:    "escaped quote in string value",
			input:   `{"message": "He said \"hello\""}`,
			want:    `{"message": "He said \"hello\""}`,
			wantErr: false,
		},
		{
			name:    "multiple escaped quotes",
			input:   `{"text": "\"quoted\" and \"again\""}`,
			want:    `{"text": "\"quoted\" and \"again\""}`,
			wantErr: false,
		},
		{
			name:    "escaped quote at string start",
			input:   `{"text": "\"start"}`,
			want:    `{"text": "\"start"}`,
			wantErr: false,
		},
		{
			name:    "escaped quote at string end",
			input:   `{"text": "end\""}`,
			want:    `{"text": "end\""}`,
			wantErr: false,
		},

		// JSON with backslashes before quotes
		{
			name:    "single backslash before quote (escaped quote)",
			input:   `{"path": "C:\\Users\\name"}`,
			want:    `{"path": "C:\\Users\\name"}`,
			wantErr: false,
		},
		{
			name:    "double backslash before quote (escaped backslash then real quote)",
			input:   `{"text": "ends with backslash\\"}`,
			want:    `{"text": "ends with backslash\\"}`,
			wantErr: false,
		},
		{
			name:    "triple backslash before quote (escaped backslash + escaped quote)",
			input:   `{"text": "backslash then quote\\\"more"}`,
			want:    `{"text": "backslash then quote\\\"more"}`,
			wantErr: false,
		},
		{
			name:    "four backslashes before quote (two escaped backslashes then real quote)",
			input:   `{"text": "double backslash\\\\"}`,
			want:    `{"text": "double backslash\\\\"}`,
			wantErr: false,
		},
		{
			name:    "mixed backslashes and quotes",
			input:   `{"code": "if (x == \"\\\\path\\\\\") {}"}`,
			want:    `{"code": "if (x == \"\\\\path\\\\\") {}"}`,
			wantErr: false,
		},

		// Braces inside strings
		{
			name:    "opening brace in string",
			input:   `{"data": "value with { brace"}`,
			want:    `{"data": "value with { brace"}`,
			wantErr: false,
		},
		{
			name:    "closing brace in string",
			input:   `{"data": "value with } brace"}`,
			want:    `{"data": "value with } brace"}`,
			wantErr: false,
		},
		{
			name:    "both braces in string",
			input:   `{"data": "contains {} braces"}`,
			want:    `{"data": "contains {} braces"}`,
			wantErr: false,
		},
		{
			name:    "nested braces in string",
			input:   `{"data": "looks like {\"nested\": true}"}`,
			want:    `{"data": "looks like {\"nested\": true}"}`,
			wantErr: false,
		},
		{
			name:    "JSON-like string value",
			input:   `{"template": "{\"type\": \"object\"}"}`,
			want:    `{"template": "{\"type\": \"object\"}"}`,
			wantErr: false,
		},

		// Complex real-world scenarios
		{
			name:    "code snippet with escaped quotes",
			input:   `{"fix": {"code": "fmt.Printf(\"value: %s\\n\", x)"}}`,
			want:    `{"fix": {"code": "fmt.Printf(\"value: %s\\n\", x)"}}`,
			wantErr: false,
		},
		{
			name:    "regex pattern in JSON",
			input:   `{"pattern": "\\d+\\.\\d+"}`,
			want:    `{"pattern": "\\d+\\.\\d+"}`,
			wantErr: false,
		},
		{
			name:    "file path with backslashes",
			input:   `{"path": "C:\\Program Files\\App\\config.json"}`,
			want:    `{"path": "C:\\Program Files\\App\\config.json"}`,
			wantErr: false,
		},
		{
			name:    "multiline string with newlines",
			input:   `{"text": "line1\nline2\nline3"}`,
			want:    `{"text": "line1\nline2\nline3"}`,
			wantErr: false,
		},
		{
			name:    "tab and special characters",
			input:   `{"text": "col1\tcol2\r\n"}`,
			want:    `{"text": "col1\tcol2\r\n"}`,
			wantErr: false,
		},

		// Edge cases with string escaping
		{
			name:    "empty string value",
			input:   `{"empty": ""}`,
			want:    `{"empty": ""}`,
			wantErr: false,
		},
		{
			name:    "string with only backslash",
			input:   `{"bs": "\\"}`,
			want:    `{"bs": "\\"}`,
			wantErr: false,
		},
		{
			name:    "string with only escaped quote",
			input:   `{"q": "\""}`,
			want:    `{"q": "\""}`,
			wantErr: false,
		},
		{
			name:    "consecutive escaped quotes",
			input:   `{"quotes": "\"\"\""}`,
			want:    `{"quotes": "\"\"\""}`,
			wantErr: false,
		},
		{
			name:    "backslash followed by non-special char",
			input:   `{"text": "\\x"}`,
			want:    `{"text": "\\x"}`,
			wantErr: false,
		},

		// Malformed JSON responses
		{
			name:    "no JSON object",
			input:   "This is just plain text with no JSON",
			want:    "",
			wantErr: true,
		},
		{
			name:    "incomplete JSON - missing closing brace",
			input:   `{"key": "value"`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "incomplete JSON - nested missing brace",
			input:   `{"outer": {"inner": "value"}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "unclosed string with brace",
			input:   `{"key": "unclosed string}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "only opening brace",
			input:   `{`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   \n\t  ",
			want:    "",
			wantErr: true,
		},

		// Review result format (realistic use case)
		{
			name:    "typical review result",
			input:   `{"mode": "security", "status": "issues_found", "summary": "Found potential SQL injection", "issues": [{"severity": "high", "description": "SQL query uses string concatenation", "location": "db.go:42", "fix": {"available": true, "code": "db.Query(\"SELECT * FROM users WHERE id = ?\", id)", "explanation": "Use parameterized query"}}]}`,
			want:    `{"mode": "security", "status": "issues_found", "summary": "Found potential SQL injection", "issues": [{"severity": "high", "description": "SQL query uses string concatenation", "location": "db.go:42", "fix": {"available": true, "code": "db.Query(\"SELECT * FROM users WHERE id = ?\", id)", "explanation": "Use parameterized query"}}]}`,
			wantErr: false,
		},
		{
			name:    "commit message result",
			input:   `{"type": "feat", "scope": "auth", "subject": "add OAuth2 login support", "body": "Implements OAuth2 flow with support for:\n- Google\n- GitHub\n- GitLab"}`,
			want:    `{"type": "feat", "scope": "auth", "subject": "add OAuth2 login support", "body": "Implements OAuth2 flow with support for:\n- Google\n- GitHub\n- GitLab"}`,
			wantErr: false,
		},

		// Arrays in JSON
		{
			name:    "JSON with array",
			input:   `{"items": ["a", "b", "c"]}`,
			want:    `{"items": ["a", "b", "c"]}`,
			wantErr: false,
		},
		{
			name:    "JSON with array of objects",
			input:   `{"items": [{"id": 1}, {"id": 2}]}`,
			want:    `{"items": [{"id": 1}, {"id": 2}]}`,
			wantErr: false,
		},
		{
			name:    "JSON with nested arrays",
			input:   `{"matrix": [[1, 2], [3, 4]]}`,
			want:    `{"matrix": [[1, 2], [3, 4]]}`,
			wantErr: false,
		},

		// Special characters
		{
			name:    "unicode characters",
			input:   `{"emoji": "Hello"}`,
			want:    `{"emoji": "Hello"}`,
			wantErr: false,
		},
		{
			name:    "unicode escape sequences",
			input:   `{"unicode": "\u0048\u0065\u006c\u006c\u006f"}`,
			want:    `{"unicode": "\u0048\u0065\u006c\u006c\u006f"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractJSON_BackslashQuoteEdgeCases specifically tests the backslash counting
// logic for escaped quotes in the JSON extraction code.
func TestExtractJSON_BackslashQuoteEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "zero backslashes before quote - ends string",
			input:   `{"a": "text"}`,
			want:    `{"a": "text"}`,
			wantErr: false,
		},
		{
			name:    "one backslash before quote - escaped quote (odd count)",
			input:   `{"a": "text\"more"}`,
			want:    `{"a": "text\"more"}`,
			wantErr: false,
		},
		{
			name:    "two backslashes before quote - escaped backslash then end string (even count)",
			input:   `{"a": "text\\"}`,
			want:    `{"a": "text\\"}`,
			wantErr: false,
		},
		{
			name:    "three backslashes before quote - escaped backslash + escaped quote (odd count)",
			input:   `{"a": "text\\\"more"}`,
			want:    `{"a": "text\\\"more"}`,
			wantErr: false,
		},
		{
			name:    "four backslashes before quote - two escaped backslashes then end string (even count)",
			input:   `{"a": "text\\\\"}`,
			want:    `{"a": "text\\\\"}`,
			wantErr: false,
		},
		{
			name:    "five backslashes before quote - two escaped backslashes + escaped quote (odd count)",
			input:   `{"a": "text\\\\\"more"}`,
			want:    `{"a": "text\\\\\"more"}`,
			wantErr: false,
		},
		{
			name:    "six backslashes before quote - three escaped backslashes then end string (even count)",
			input:   `{"a": "text\\\\\\"}`,
			want:    `{"a": "text\\\\\\"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractJSON_BracesInStrings tests that braces inside strings don't affect depth counting
func TestExtractJSON_BracesInStrings(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "opening brace in value",
			input:   `{"code": "if (x) {"}`,
			want:    `{"code": "if (x) {"}`,
			wantErr: false,
		},
		{
			name:    "closing brace in value",
			input:   `{"code": "} // end"}`,
			want:    `{"code": "} // end"}`,
			wantErr: false,
		},
		{
			name:    "unbalanced braces in string (more opens)",
			input:   `{"code": "{{{"}`,
			want:    `{"code": "{{{"}`,
			wantErr: false,
		},
		{
			name:    "unbalanced braces in string (more closes)",
			input:   `{"code": "}}}"}`,
			want:    `{"code": "}}}"}`,
			wantErr: false,
		},
		{
			name:    "braces with escaped quotes",
			input:   `{"code": "obj = {\"key\": \"value\"}"}`,
			want:    `{"code": "obj = {\"key\": \"value\"}"}`,
			wantErr: false,
		},
		{
			name:    "deeply nested braces in string",
			input:   `{"template": "{{{{nested}}}}"}`,
			want:    `{"template": "{{{{nested}}}}"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractJSON_MarkdownCodeBlocks tests extraction from markdown-wrapped JSON
func TestExtractJSON_MarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "json code block",
			input:   "```json\n{\"key\": \"value\"}\n```",
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "plain code block",
			input:   "```\n{\"key\": \"value\"}\n```",
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "code block with extra whitespace",
			input:   "```json\n\n  {\"key\": \"value\"}\n\n```",
			want:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "code block without trailing newline",
			input:   "```json\n{\"key\": \"value\"}```",
			want:    `{"key": "value"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTruncateDiff tests the diff truncation function
func TestTruncateDiff(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		truncate bool
	}{
		{
			name:     "short diff unchanged",
			input:    "short diff",
			wantLen:  10,
			truncate: false,
		},
		{
			name:     "exactly at limit unchanged",
			input:    strings.Repeat("a", MaxDiffSize),
			wantLen:  MaxDiffSize,
			truncate: false,
		},
		{
			name:     "over limit gets truncated",
			input:    strings.Repeat("a", MaxDiffSize+100),
			wantLen:  MaxDiffSize + len("\n\n[... diff truncated due to size limits ...]"),
			truncate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateDiff(tt.input)
			if tt.truncate {
				if !strings.HasSuffix(got, "[... diff truncated due to size limits ...]") {
					t.Error("truncated diff should have truncation message")
				}
			} else {
				if got != tt.input {
					t.Errorf("non-truncated diff should be unchanged, got len %d", len(got))
				}
			}
		})
	}
}

// TestCommitMessageString tests the CommitMessage.String() method
func TestCommitMessageString(t *testing.T) {
	tests := []struct {
		name string
		msg  CommitMessage
		want string
	}{
		{
			name: "simple message without scope",
			msg: CommitMessage{
				Type:    "feat",
				Subject: "add new feature",
			},
			want: "feat: add new feature",
		},
		{
			name: "message with scope",
			msg: CommitMessage{
				Type:    "fix",
				Scope:   "auth",
				Subject: "fix login bug",
			},
			want: "fix(auth): fix login bug",
		},
		{
			name: "message with body",
			msg: CommitMessage{
				Type:    "feat",
				Subject: "add feature",
				Body:    "This is a longer description.",
			},
			want: "feat: add feature\n\nThis is a longer description.",
		},
		{
			name: "message with scope and body",
			msg: CommitMessage{
				Type:    "refactor",
				Scope:   "api",
				Subject: "restructure endpoints",
				Body:    "Reorganized API structure for clarity.",
			},
			want: "refactor(api): restructure endpoints\n\nReorganized API structure for clarity.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.String()
			if got != tt.want {
				t.Errorf("CommitMessage.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNewClient tests the NewClient constructor
func TestNewClient(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		timeout        int
		wantPath       string
		wantTimeoutSec int
	}{
		{
			name:           "default values",
			path:           "",
			timeout:        0,
			wantPath:       "claude",
			wantTimeoutSec: 60,
		},
		{
			name:           "custom path",
			path:           "/usr/local/bin/claude",
			timeout:        0,
			wantPath:       "/usr/local/bin/claude",
			wantTimeoutSec: 60,
		},
		{
			name:           "custom timeout",
			path:           "",
			timeout:        120,
			wantPath:       "claude",
			wantTimeoutSec: 120,
		},
		{
			name:           "negative timeout uses default",
			path:           "",
			timeout:        -1,
			wantPath:       "claude",
			wantTimeoutSec: 60,
		},
		{
			name:           "all custom",
			path:           "/custom/claude",
			timeout:        300,
			wantPath:       "/custom/claude",
			wantTimeoutSec: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.path, tt.timeout)
			if client.path != tt.wantPath {
				t.Errorf("NewClient().path = %q, want %q", client.path, tt.wantPath)
			}
			wantDuration := tt.wantTimeoutSec * int(1e9) // convert to nanoseconds
			if int(client.timeout) != wantDuration {
				t.Errorf("NewClient().timeout = %v, want %v seconds", client.timeout, tt.wantTimeoutSec)
			}
		})
	}
}
