// Package commit provides utilities for generating and validating conventional
// commit messages. It wraps the Claude client for AI-powered message generation
// and includes helpers for parsing and formatting commit messages.
package commit

import (
	"context"
	"fmt"
	"strings"

	"github.com/buker/revi/internal/claude"
)

// Generator handles AI-powered commit message generation using Claude.
type Generator struct {
	client *claude.Client
}

// NewGenerator creates a new Generator with the given Claude client.
func NewGenerator(client *claude.Client) *Generator {
	return &Generator{client: client}
}

// Generate creates a conventional commit message by analyzing the given git diff.
// Returns a structured CommitMessage with type, optional scope, subject, and body.
func (g *Generator) Generate(ctx context.Context, diff string) (*claude.CommitMessage, error) {
	return g.client.GenerateCommitMessage(ctx, diff)
}

// FormatMessage formats a CommitMessage as a string suitable for git commit.
func FormatMessage(msg *claude.CommitMessage) string {
	return msg.String()
}

// ValidateMessage validates a CommitMessage against conventional commit rules.
// Returns an error if the type is invalid, subject is missing, or subject exceeds 50 chars.
func ValidateMessage(msg *claude.CommitMessage) error {
	if msg.Type == "" {
		return fmt.Errorf("commit type is required")
	}

	validTypes := []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "chore"}
	isValid := false
	for _, t := range validTypes {
		if msg.Type == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid commit type: %s", msg.Type)
	}

	if msg.Subject == "" {
		return fmt.Errorf("commit subject is required")
	}

	if len(msg.Subject) > 50 {
		return fmt.Errorf("commit subject too long: %d chars (max 50)", len(msg.Subject))
	}

	return nil
}

// ParseMessage parses a formatted conventional commit string back into a CommitMessage.
// It extracts the type, optional scope, subject, and body from the message.
func ParseMessage(message string) (*claude.CommitMessage, error) {
	lines := strings.SplitN(message, "\n", 2)
	if len(lines) == 0 || lines[0] == "" {
		return nil, fmt.Errorf("empty commit message")
	}

	firstLine := lines[0]
	msg := &claude.CommitMessage{}

	// Parse type(scope): subject
	colonIdx := strings.Index(firstLine, ":")
	if colonIdx == -1 {
		return nil, fmt.Errorf("invalid commit format: missing colon")
	}

	typeScope := firstLine[:colonIdx]
	msg.Subject = strings.TrimSpace(firstLine[colonIdx+1:])

	// Check for scope in parentheses
	parenOpen := strings.Index(typeScope, "(")
	parenClose := strings.Index(typeScope, ")")

	if parenOpen != -1 && parenClose != -1 && parenClose > parenOpen {
		msg.Type = typeScope[:parenOpen]
		msg.Scope = typeScope[parenOpen+1 : parenClose]
	} else {
		msg.Type = typeScope
	}

	// Parse body if present
	if len(lines) > 1 {
		msg.Body = strings.TrimSpace(lines[1])
	}

	return msg, nil
}

// TypeDescription returns a human-readable description for a conventional commit type.
// Returns an empty string for unknown types.
func TypeDescription(commitType string) string {
	descriptions := map[string]string{
		"feat":     "A new feature",
		"fix":      "A bug fix",
		"docs":     "Documentation only changes",
		"style":    "Changes that do not affect the meaning of the code",
		"refactor": "A code change that neither fixes a bug nor adds a feature",
		"perf":     "A code change that improves performance",
		"test":     "Adding missing tests or correcting existing tests",
		"chore":    "Changes to the build process or auxiliary tools",
	}
	return descriptions[commitType]
}
