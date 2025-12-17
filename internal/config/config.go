// Package config manages application configuration using viper.
// It supports configuration from YAML files (.revi.yaml), environment variables
// (REVI_ prefix), and command-line flags with sensible defaults.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all application configuration values.
// It is populated from config files, environment variables, and command-line flags.
type Config struct {
	Review ReviewConfig `mapstructure:"review"` // Review behavior settings
	Commit CommitConfig `mapstructure:"commit"` // Commit generation settings
	AI     AIConfig     `mapstructure:"ai"`     // AI provider settings
}

// ReviewConfig holds configuration for code review behavior.
type ReviewConfig struct {
	Enabled bool        `mapstructure:"enabled"` // Whether to run code review
	Block   bool        `mapstructure:"block"`   // Whether to block commits on high-severity issues
	Modes   ReviewModes `mapstructure:"modes"`   // Individual mode toggles
}

// ReviewModes holds on/off settings for each review mode.
// When a mode is disabled, it will be skipped during review.
type ReviewModes struct {
	Security    bool `mapstructure:"security"`    // Check for security vulnerabilities
	Performance bool `mapstructure:"performance"` // Check for performance issues
	Style       bool `mapstructure:"style"`       // Check code style and patterns
	Errors      bool `mapstructure:"errors"`      // Check error handling
	Testing     bool `mapstructure:"testing"`     // Check test coverage
	Docs        bool `mapstructure:"docs"`        // Check documentation
}

// CommitConfig holds configuration for commit message generation.
type CommitConfig struct {
	Enabled bool `mapstructure:"enabled"` // Whether to generate commit messages
}

// AIConfig holds configuration for the AI provider integration.
// The model can be overridden via REVI_AI_MODEL environment variable or --model flag.
type AIConfig struct {
	Model string `mapstructure:"model"` // AI model identifier (e.g., claude-opus-4-5-20251101)
}

var (
	cfg        Config
	configFile string
)

// Init initializes the configuration system by setting defaults,
// loading config files from current and home directories, and
// enabling environment variable overrides with the REVI_ prefix.
func Init() {
	setDefaults()
	loadConfigFile()
	loadEnvVars()
}

func setDefaults() {
	// Review defaults
	viper.SetDefault("review.enabled", true)
	viper.SetDefault("review.block", true)
	viper.SetDefault("review.modes.security", true)
	viper.SetDefault("review.modes.performance", true)
	viper.SetDefault("review.modes.style", true)
	viper.SetDefault("review.modes.errors", true)
	viper.SetDefault("review.modes.testing", true)
	viper.SetDefault("review.modes.docs", true)

	// Commit defaults
	viper.SetDefault("commit.enabled", true)

	// AI defaults - uses Claude Opus 4.5 as the default model
	viper.SetDefault("ai.model", "claude-opus-4-5-20251101")
}

func loadConfigFile() {
	viper.SetConfigName(".revi")
	viper.SetConfigType("yaml")

	// Add config paths in priority order
	// 1. Current directory (project config)
	viper.AddConfigPath(".")
	// 2. Home directory (global config)
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(home)
	}

	if err := viper.ReadInConfig(); err == nil {
		configFile = viper.ConfigFileUsed()
	}
}

func loadEnvVars() {
	viper.SetEnvPrefix("REVI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// BindFlags binds cobra command-line flags to viper configuration values.
// This enables flags like --review, --block, and --model to override config file settings.
func BindFlags(cmd *cobra.Command) {
	// Bind persistent flags - errors are ignored as flags are guaranteed to exist
	_ = viper.BindPFlag("review.enabled", cmd.PersistentFlags().Lookup("review"))
	_ = viper.BindPFlag("review.block", cmd.PersistentFlags().Lookup("block"))

	// Review mode flags
	_ = viper.BindPFlag("review.modes.security", cmd.PersistentFlags().Lookup("security"))
	_ = viper.BindPFlag("review.modes.performance", cmd.PersistentFlags().Lookup("performance"))
	_ = viper.BindPFlag("review.modes.style", cmd.PersistentFlags().Lookup("style"))
	_ = viper.BindPFlag("review.modes.errors", cmd.PersistentFlags().Lookup("errors"))
	_ = viper.BindPFlag("review.modes.testing", cmd.PersistentFlags().Lookup("testing"))
	_ = viper.BindPFlag("review.modes.docs", cmd.PersistentFlags().Lookup("docs"))

	// AI model flag
	_ = viper.BindPFlag("ai.model", cmd.PersistentFlags().Lookup("model"))
}

// Get returns the current configuration by unmarshaling all viper values.
// Call this after Init and BindFlags to get the final merged configuration.
func Get() *Config {
	// Error is ignored as defaults are always valid
	_ = viper.Unmarshal(&cfg)
	return &cfg
}

// GetConfigPath returns the path to the config file that was loaded,
// or an empty string if no config file was found.
func GetConfigPath() string {
	return configFile
}

// GetDefaultConfigPath returns the default global config file path (~/.revi.yaml).
func GetDefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".revi.yaml")
}

// IsReviewEnabled checks if code review is enabled, considering both the
// --no-review flag and the review.enabled config setting.
func IsReviewEnabled(cmd *cobra.Command) bool {
	noReview, _ := cmd.Flags().GetBool("no-review")
	if noReview {
		return false
	}
	return viper.GetBool("review.enabled")
}

// IsBlockEnabled checks if commit blocking is enabled, considering both the
// --no-block flag and the review.block config setting.
func IsBlockEnabled(cmd *cobra.Command) bool {
	noBlock, _ := cmd.Flags().GetBool("no-block")
	if noBlock {
		return false
	}
	return viper.GetBool("review.block")
}

// GetEnabledModes returns the list of review modes that should be run.
// It respects the --all flag, individual --no-<mode> flags, and config settings.
func GetEnabledModes(cmd *cobra.Command) []string {
	// Check if --all flag is set
	allModes, _ := cmd.Flags().GetBool("all")

	modes := []string{}
	cfg := Get()

	checkMode := func(name string, enabled bool, noFlagName string) {
		if allModes {
			modes = append(modes, name)
			return
		}
		noFlag, _ := cmd.Flags().GetBool(noFlagName)
		if !noFlag && enabled {
			modes = append(modes, name)
		}
	}

	checkMode("security", cfg.Review.Modes.Security, "no-security")
	checkMode("performance", cfg.Review.Modes.Performance, "no-performance")
	checkMode("style", cfg.Review.Modes.Style, "no-style")
	checkMode("errors", cfg.Review.Modes.Errors, "no-errors")
	checkMode("testing", cfg.Review.Modes.Testing, "no-testing")
	checkMode("docs", cfg.Review.Modes.Docs, "no-docs")

	return modes
}
