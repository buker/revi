package commit

import (
	"strings"
	"testing"

	"github.com/buker/revi/internal/ai"
)

func TestValidateMessage(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		err := ValidateMessage(&ai.CommitMessage{Type: "feat", Subject: "add staging diff support"})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("missing type", func(t *testing.T) {
		err := ValidateMessage(&ai.CommitMessage{Subject: "add thing"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		err := ValidateMessage(&ai.CommitMessage{Type: "feature", Subject: "add thing"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing subject", func(t *testing.T) {
		err := ValidateMessage(&ai.CommitMessage{Type: "fix"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("subject too long", func(t *testing.T) {
		long := strings.Repeat("a", 51)
		err := ValidateMessage(&ai.CommitMessage{Type: "chore", Subject: long})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseMessage(t *testing.T) {
	t.Run("type only", func(t *testing.T) {
		msg, err := ParseMessage("feat: add new feature")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.Type != "feat" {
			t.Fatalf("expected type feat, got %q", msg.Type)
		}
		if msg.Scope != "" {
			t.Fatalf("expected empty scope, got %q", msg.Scope)
		}
		if msg.Subject != "add new feature" {
			t.Fatalf("expected subject %q, got %q", "add new feature", msg.Subject)
		}
		if msg.Body != "" {
			t.Fatalf("expected empty body, got %q", msg.Body)
		}
	})

	t.Run("type and scope with body", func(t *testing.T) {
		input := "fix(auth): handle missing token\n\nAdds better error messages."
		msg, err := ParseMessage(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.Type != "fix" {
			t.Fatalf("expected type fix, got %q", msg.Type)
		}
		if msg.Scope != "auth" {
			t.Fatalf("expected scope auth, got %q", msg.Scope)
		}
		if msg.Subject != "handle missing token" {
			t.Fatalf("expected subject %q, got %q", "handle missing token", msg.Subject)
		}
		if msg.Body != "Adds better error messages." {
			t.Fatalf("expected body %q, got %q", "Adds better error messages.", msg.Body)
		}
	})

	t.Run("missing colon", func(t *testing.T) {
		_, err := ParseMessage("feat add thing")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty message", func(t *testing.T) {
		_, err := ParseMessage("")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestFormatMessage(t *testing.T) {
	msg := &ai.CommitMessage{Type: "feat", Scope: "cli", Subject: "add config subcommand", Body: "Provides config show/path."}
	got := FormatMessage(msg)
	want := "feat(cli): add config subcommand\n\nProvides config show/path."
	if got != want {
		t.Fatalf("FormatMessage() = %q, want %q", got, want)
	}
}

func TestTypeDescription(t *testing.T) {
	if TypeDescription("feat") == "" {
		t.Fatal("expected non-empty description for feat")
	}
	if TypeDescription("unknown") != "" {
		t.Fatal("expected empty description for unknown type")
	}
}
