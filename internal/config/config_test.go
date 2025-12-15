package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func resetForTest(t *testing.T) {
	t.Helper()
	viper.Reset()
	cfg = Config{}
	configFile = ""
	// Prevent accidentally reading a real user config from HOME.
	t.Setenv("HOME", t.TempDir())
}

func newCmdForEnabledModesTest() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().Bool("no-security", false, "")
	cmd.Flags().Bool("no-performance", false, "")
	cmd.Flags().Bool("no-style", false, "")
	cmd.Flags().Bool("no-errors", false, "")
	cmd.Flags().Bool("no-testing", false, "")
	cmd.Flags().Bool("no-docs", false, "")
	return cmd
}

func TestInit_SetsDefaults(t *testing.T) {
	resetForTest(t)
	Init()

	c := Get()
	if !c.Review.Enabled {
		t.Fatal("expected review.enabled default to be true")
	}
	if !c.Review.Block {
		t.Fatal("expected review.block default to be true")
	}
	if !c.Commit.Enabled {
		t.Fatal("expected commit.enabled default to be true")
	}
	if c.Claude.Path != "claude" {
		t.Fatalf("expected claude.path default %q, got %q", "claude", c.Claude.Path)
	}
	if c.Claude.Timeout != 60 {
		t.Fatalf("expected claude.timeout default %d, got %d", 60, c.Claude.Timeout)
	}

	if GetConfigPath() != "" {
		t.Fatalf("expected no config file to be loaded in tests, got %q", GetConfigPath())
	}
}

func TestInit_EnvOverrides(t *testing.T) {
	resetForTest(t)
	t.Setenv("REVI_CLAUDE_PATH", "/custom/claude")
	t.Setenv("REVI_CLAUDE_TIMEOUT", "120")
	t.Setenv("REVI_REVIEW_BLOCK", "false")

	Init()
	c := Get()

	if c.Claude.Path != "/custom/claude" {
		t.Fatalf("expected claude.path override %q, got %q", "/custom/claude", c.Claude.Path)
	}
	if c.Claude.Timeout != 120 {
		t.Fatalf("expected claude.timeout override %d, got %d", 120, c.Claude.Timeout)
	}
	if c.Review.Block != false {
		t.Fatalf("expected review.block override false, got %v", c.Review.Block)
	}
}

func TestGetEnabledModes_All(t *testing.T) {
	resetForTest(t)
	Init()

	cmd := newCmdForEnabledModesTest()
	_ = cmd.Flags().Set("all", "true")

	modes := GetEnabledModes(cmd)
	if len(modes) != 6 {
		t.Fatalf("expected 6 modes when --all is set, got %d (%v)", len(modes), modes)
	}
}

func TestGetEnabledModes_RespectsNoFlags(t *testing.T) {
	resetForTest(t)
	Init()

	cmd := newCmdForEnabledModesTest()
	_ = cmd.Flags().Set("no-security", "true")
	_ = cmd.Flags().Set("no-docs", "true")

	modes := GetEnabledModes(cmd)
	for _, m := range modes {
		if m == "security" {
			t.Fatal("expected security to be omitted when --no-security is set")
		}
		if m == "docs" {
			t.Fatal("expected docs to be omitted when --no-docs is set")
		}
	}
}

func TestGetDefaultConfigPath_UsesHome(t *testing.T) {
	resetForTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := GetDefaultConfigPath()
	if p == "" {
		t.Fatal("expected non-empty default config path")
	}
	if p != home+"/.revi.yaml" {
		t.Fatalf("expected %q, got %q", home+"/.revi.yaml", p)
	}
}
