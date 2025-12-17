package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DefaultModel is the expected default model for tests.
const testDefaultModel = "claude-opus-4-5-20251101"

func TestAIConfig_DefaultModel(t *testing.T) {
	resetForTest(t)
	Init()

	c := Get()
	if c.AI.Model != testDefaultModel {
		t.Fatalf("expected ai.model default %q, got %q", testDefaultModel, c.AI.Model)
	}
}

func TestAIConfig_ModelOverrideFromEnv(t *testing.T) {
	resetForTest(t)
	t.Setenv("REVI_AI_MODEL", "claude-sonnet-4-20250514")

	Init()
	c := Get()

	if c.AI.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("expected ai.model env override %q, got %q", "claude-sonnet-4-20250514", c.AI.Model)
	}
}

func TestAIConfig_ModelOverrideFromYAML(t *testing.T) {
	resetForTest(t)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".revi.yaml")
	configContent := `ai:
  model: claude-haiku-3-5-20241022
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Change to temp dir so viper finds the config
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("failed to restore directory: %v", err)
		}
	})

	Init()
	c := Get()

	if c.AI.Model != "claude-haiku-3-5-20241022" {
		t.Fatalf("expected ai.model from YAML %q, got %q", "claude-haiku-3-5-20241022", c.AI.Model)
	}
}

func TestAIConfig_ModelFlagOverride(t *testing.T) {
	resetForTest(t)
	Init()

	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().String("model", "", "AI model to use")
	_ = viper.BindPFlag("ai.model", cmd.PersistentFlags().Lookup("model"))
	_ = cmd.PersistentFlags().Set("model", "claude-sonnet-4-20250514")

	c := Get()
	if c.AI.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("expected ai.model flag override %q, got %q", "claude-sonnet-4-20250514", c.AI.Model)
	}
}

func TestAIConfig_ThreeTierPriority(t *testing.T) {
	resetForTest(t)

	// Set up YAML config (lowest priority)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".revi.yaml")
	configContent := `ai:
  model: model-from-yaml
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("failed to restore directory: %v", err)
		}
	})

	// Set env var (middle priority)
	t.Setenv("REVI_AI_MODEL", "model-from-env")

	Init()

	// Verify env overrides YAML
	c := Get()
	if c.AI.Model != "model-from-env" {
		t.Fatalf("expected env to override YAML, got %q", c.AI.Model)
	}

	// Set up flag (highest priority)
	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().String("model", "", "AI model to use")
	_ = viper.BindPFlag("ai.model", cmd.PersistentFlags().Lookup("model"))
	_ = cmd.PersistentFlags().Set("model", "model-from-flag")

	// Verify flag overrides env
	c = Get()
	if c.AI.Model != "model-from-flag" {
		t.Fatalf("expected flag to override env, got %q", c.AI.Model)
	}
}

func TestAIConfig_NoClaudePathOrTimeout(t *testing.T) {
	resetForTest(t)
	Init()

	// Verify old claude.path and claude.timeout are not set
	if viper.IsSet("claude.path") {
		t.Error("claude.path should not be set in new config")
	}
	if viper.IsSet("claude.timeout") {
		t.Error("claude.timeout should not be set in new config")
	}
}
