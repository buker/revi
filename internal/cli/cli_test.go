package cli

import (
	"testing"

	"github.com/buker/revi/internal/review"
	"github.com/spf13/cobra"
)

// =============================================================================
// Tests for shortHash function
// =============================================================================

func TestShortHash_NormalHash(t *testing.T) {
	hash := "a1b2c3d4e5f67890123456789abcdef0123456789"
	result := shortHash(hash)
	if result != "a1b2c3d4" {
		t.Errorf("expected %q, got %q", "a1b2c3d4", result)
	}
}

func TestShortHash_ExactlyEightChars(t *testing.T) {
	hash := "a1b2c3d4"
	result := shortHash(hash)
	if result != "a1b2c3d4" {
		t.Errorf("expected %q, got %q", "a1b2c3d4", result)
	}
}

func TestShortHash_ShorterThanEight(t *testing.T) {
	hash := "a1b2c3"
	result := shortHash(hash)
	if result != "a1b2c3" {
		t.Errorf("expected %q, got %q", "a1b2c3", result)
	}
}

func TestShortHash_EmptyString(t *testing.T) {
	hash := ""
	result := shortHash(hash)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// =============================================================================
// Tests for filterModesByFlags function
// =============================================================================

func newReviewCmdForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "review"}
	// Enabled flags
	cmd.Flags().Bool("security", false, "")
	cmd.Flags().Bool("performance", false, "")
	cmd.Flags().Bool("style", false, "")
	cmd.Flags().Bool("errors", false, "")
	cmd.Flags().Bool("testing", false, "")
	cmd.Flags().Bool("docs", false, "")
	// Disabled flags
	cmd.Flags().Bool("no-security", false, "")
	cmd.Flags().Bool("no-performance", false, "")
	cmd.Flags().Bool("no-style", false, "")
	cmd.Flags().Bool("no-errors", false, "")
	cmd.Flags().Bool("no-testing", false, "")
	cmd.Flags().Bool("no-docs", false, "")
	return cmd
}

func TestFilterModesByFlags_NoFlagsSet(t *testing.T) {
	cmd := newReviewCmdForTest()
	detected := []review.Mode{review.ModeSecurity, review.ModeStyle}

	result := filterModesByFlags(cmd, detected)

	if len(result) != 2 {
		t.Errorf("expected 2 modes, got %d", len(result))
	}
}

func TestFilterModesByFlags_DisableOneMode(t *testing.T) {
	cmd := newReviewCmdForTest()
	_ = cmd.Flags().Set("no-security", "true")
	detected := []review.Mode{review.ModeSecurity, review.ModeStyle, review.ModeErrors}

	result := filterModesByFlags(cmd, detected)

	// Security should be removed
	for _, mode := range result {
		if mode == review.ModeSecurity {
			t.Error("security mode should be filtered out")
		}
	}
	if len(result) != 2 {
		t.Errorf("expected 2 modes after filtering, got %d", len(result))
	}
}

func TestFilterModesByFlags_EnableModeNotInDetected(t *testing.T) {
	cmd := newReviewCmdForTest()
	_ = cmd.Flags().Set("docs", "true")
	detected := []review.Mode{review.ModeSecurity}

	result := filterModesByFlags(cmd, detected)

	// Docs should be added since it was explicitly enabled
	hasDocs := false
	for _, mode := range result {
		if mode == review.ModeDocs {
			hasDocs = true
		}
	}
	if !hasDocs {
		t.Error("docs mode should be added when explicitly enabled")
	}
}

func TestFilterModesByFlags_DisableMultipleModes(t *testing.T) {
	cmd := newReviewCmdForTest()
	_ = cmd.Flags().Set("no-security", "true")
	_ = cmd.Flags().Set("no-performance", "true")
	_ = cmd.Flags().Set("no-docs", "true")

	detected := review.AllModes()
	result := filterModesByFlags(cmd, detected)

	// Should have 3 modes (6 total - 3 disabled)
	if len(result) != 3 {
		t.Errorf("expected 3 modes after filtering, got %d: %v", len(result), result)
	}

	// Verify disabled modes are not present
	disabledModes := map[review.Mode]bool{
		review.ModeSecurity:    true,
		review.ModePerformance: true,
		review.ModeDocs:        true,
	}
	for _, mode := range result {
		if disabledModes[mode] {
			t.Errorf("mode %s should be filtered out", mode)
		}
	}
}

// =============================================================================
// Tests for isBlockEnabled function
// =============================================================================

func newBlockCmdForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "review"}
	cmd.Flags().BoolP("block", "b", true, "")
	cmd.Flags().BoolP("no-block", "B", false, "")
	return cmd
}

func TestIsBlockEnabled_DefaultTrue(t *testing.T) {
	cmd := newBlockCmdForTest()
	result := isBlockEnabled(cmd)
	if !result {
		t.Error("expected block to be enabled by default")
	}
}

func TestIsBlockEnabled_NoBlockFlag(t *testing.T) {
	cmd := newBlockCmdForTest()
	_ = cmd.Flags().Set("no-block", "true")
	result := isBlockEnabled(cmd)
	if result {
		t.Error("expected block to be disabled when --no-block is set")
	}
}

func TestIsBlockEnabled_BlockFalse(t *testing.T) {
	cmd := newBlockCmdForTest()
	_ = cmd.Flags().Set("block", "false")
	result := isBlockEnabled(cmd)
	if result {
		t.Error("expected block to be disabled when --block=false")
	}
}

func TestIsBlockEnabled_NoBlockOverridesBlock(t *testing.T) {
	cmd := newBlockCmdForTest()
	_ = cmd.Flags().Set("block", "true")
	_ = cmd.Flags().Set("no-block", "true")
	result := isBlockEnabled(cmd)
	if result {
		t.Error("--no-block should override --block")
	}
}

// =============================================================================
// Tests for root command structure
// =============================================================================

func TestRootCmd_HasExpectedSubcommands(t *testing.T) {
	subcommands := rootCmd.Commands()
	expected := map[string]bool{
		"review":  false,
		"commit":  false,
		"config":  false,
		"version": false,
	}

	for _, cmd := range subcommands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestRootCmd_HasDryRunFlag(t *testing.T) {
	flag := rootCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Error("expected --dry-run flag on root command")
	}
	if flag.Shorthand != "n" {
		t.Errorf("expected shorthand 'n' for dry-run, got %q", flag.Shorthand)
	}
}

func TestRootCmd_HasMessageFlag(t *testing.T) {
	flag := rootCmd.Flags().Lookup("message")
	if flag == nil {
		t.Error("expected --message flag on root command")
	}
	if flag.Shorthand != "m" {
		t.Errorf("expected shorthand 'm' for message, got %q", flag.Shorthand)
	}
}

// =============================================================================
// Tests for version command
// =============================================================================

func TestVersionCmd_HasCorrectUse(t *testing.T) {
	if versionCmd.Use != "version" {
		t.Errorf("expected Use to be 'version', got %q", versionCmd.Use)
	}
}

func TestVersionCmd_HasShortDescription(t *testing.T) {
	if versionCmd.Short == "" {
		t.Error("expected version command to have a short description")
	}
}

func TestVersionCmd_DoesNotPanic(t *testing.T) {
	// Save original version
	origVersion := Version
	defer func() { Version = origVersion }()

	Version = "test-version"

	// Run the command directly - this verifies it doesn't panic
	// Output goes to stdout which we don't capture in this test
	versionCmd.Run(versionCmd, []string{})
}


// =============================================================================
// Tests for commit command structure
// =============================================================================

func TestCommitCmd_HasSameFlags(t *testing.T) {
	// Commit command should have the same flags as root
	dryRun := commitCmd.Flags().Lookup("dry-run")
	if dryRun == nil {
		t.Error("expected --dry-run flag on commit command")
	}

	message := commitCmd.Flags().Lookup("message")
	if message == nil {
		t.Error("expected --message flag on commit command")
	}
}

// =============================================================================
// Tests for review command structure
// =============================================================================

func TestReviewCmd_HasFixFlag(t *testing.T) {
	flag := reviewCmd.Flags().Lookup("fix")
	if flag == nil {
		t.Error("expected --fix flag on review command")
	}
	if flag.Shorthand != "f" {
		t.Errorf("expected shorthand 'f' for fix, got %q", flag.Shorthand)
	}
}

func TestReviewCmd_HasBlockFlags(t *testing.T) {
	block := reviewCmd.Flags().Lookup("block")
	if block == nil {
		t.Error("expected --block flag on review command")
	}

	noBlock := reviewCmd.Flags().Lookup("no-block")
	if noBlock == nil {
		t.Error("expected --no-block flag on review command")
	}
}

func TestReviewCmd_HasAllModeFlags(t *testing.T) {
	modeFlags := []string{
		"security", "no-security",
		"performance", "no-performance",
		"style", "no-style",
		"errors", "no-errors",
		"testing", "no-testing",
		"docs", "no-docs",
		"all",
	}

	for _, flagName := range modeFlags {
		flag := reviewCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected --%s flag on review command", flagName)
		}
	}
}

// =============================================================================
// Tests for config command structure
// =============================================================================

func TestConfigCmd_HasSubcommands(t *testing.T) {
	subcommands := configCmd.Commands()
	expected := map[string]bool{
		"show": false,
		"path": false,
	}

	for _, cmd := range subcommands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected config subcommand %q not found", name)
		}
	}
}

