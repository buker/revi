// Package cli implements the command-line interface for revi using cobra.
// It provides commands for commit message generation and standalone code review.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	claudecode "github.com/rokrokss/claude-code-sdk-go"

	"github.com/buker/revi/internal/ai"
	"github.com/buker/revi/internal/config"
	"github.com/buker/revi/internal/git"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version is set at build time via -ldflags
	Version = "dev"

	// debug controls debug logging output
	debug bool

	rootCmd = &cobra.Command{
		Use:   "revi",
		Short: "AI-powered commit message generator",
		Long: `revi generates AI-powered commit messages for staged changes.

Usage:
  revi           Generate commit message and commit
  revi review    Run AI code reviews on staged changes`,
		RunE: runFullWorkflow,
	}
)

func init() {
	cobra.OnInitialize(config.Init)

	// Persistent flags available to all commands
	rootCmd.PersistentFlags().String("model", "", "AI model to use (default: claude-sonnet-4-20250514)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	// Root command flags
	rootCmd.Flags().BoolP("dry-run", "n", false, "Preview commit message without committing")
	rootCmd.Flags().StringP("message", "m", "", "Context explaining why this change was made")

	// Bind persistent flags to viper
	_ = viper.BindPFlag("ai.model", rootCmd.PersistentFlags().Lookup("model"))

	// Add subcommands
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// debugLog prints a debug message if debug mode is enabled
func debugLog(format string, args ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Execute runs the root command and returns any error encountered.
// This is the main entry point for the CLI application.
func Execute() error {
	return rootCmd.Execute()
}

func runFullWorkflow(cmd *cobra.Command, args []string) error {
	debugLog("Starting runFullWorkflow")
	ctx := context.Background()
	cfg := config.Get()
	debugLog("Config loaded: model=%s", cfg.AI.Model)

	// Initialize AI client wrapper with model configuration
	debugLog("Initializing AI client...")
	aiClient, err := ai.NewClient(cfg.AI.Model)
	if err != nil {
		return fmt.Errorf("failed to initialize AI client: %w", err)
	}
	debugLog("AI client initialized")

	// Open git repository
	debugLog("Opening git repository...")
	repo, err := git.OpenCurrent()
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}
	debugLog("Git repository opened")

	// Check for staged changes
	debugLog("Checking for staged changes...")
	hasStagedChanges, err := repo.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}
	if !hasStagedChanges {
		return fmt.Errorf("no staged changes found. Use 'git add' to stage files")
	}
	debugLog("Staged changes found")

	// Get staged diff
	debugLog("Getting staged diff...")
	diff, err := repo.GetStagedDiff()
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %w", err)
	}
	debugLog("Staged diff retrieved (length: %d bytes)", len(diff))

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	userContext, _ := cmd.Flags().GetString("message")

	fmt.Println("Generating commit message...")

	// Use WithClient pattern to manage SDK client lifecycle
	// Single subprocess spawned for entire workflow, automatically cleaned up
	var commitMessage string
	debugLog("Calling aiClient.RunWithClient...")
	err = aiClient.RunWithClient(ctx, func(client claudecode.Client) error {
		debugLog("Inside RunWithClient callback")
		// Generate commit message with connected client
		debugLog("Calling GenerateCommitMessage...")
		msg, err := aiClient.GenerateCommitMessage(ctx, client, diff, userContext)
		if err != nil {
			debugLog("GenerateCommitMessage error: %v", err)
			return fmt.Errorf("failed to generate commit message: %w", err)
		}
		debugLog("GenerateCommitMessage succeeded")
		commitMessage = msg.String()
		debugLog("Commit message: %s", commitMessage)
		return nil
	})

	if err != nil {
		debugLog("RunWithClient returned error: %v", err)
		return err
	}
	debugLog("RunWithClient completed successfully")

	// Display commit message
	fmt.Println()
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Commit message:")
	fmt.Println()
	fmt.Println("  " + strings.ReplaceAll(commitMessage, "\n", "\n  "))
	fmt.Println()
	fmt.Println(strings.Repeat("-", 40))

	// Ask for confirmation
	fmt.Print("\nProceed with commit? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Commit cancelled.")
		return nil
	}

	if dryRun {
		fmt.Println("Dry run - commit not created.")
		return nil
	}

	// Create the commit
	hash, err := repo.Commit(commitMessage)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	fmt.Printf("Created commit: %s\n", shortHash(hash))
	return nil
}

// shortHash returns a shortened version of a git hash (first 8 chars).
// Returns the full hash if it's shorter than 8 characters.
func shortHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("revi version %s\n", Version)
	},
}
