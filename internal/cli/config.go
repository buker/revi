package cli

import (
	"fmt"

	"github.com/buker/revi/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `View and manage revi configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		fmt.Println("Current configuration:")
		fmt.Println("----------------------")
		fmt.Printf("Review enabled:  %v\n", cfg.Review.Enabled)
		fmt.Printf("Review block:    %v\n", cfg.Review.Block)
		fmt.Printf("Commit enabled:  %v\n", cfg.Commit.Enabled)
		fmt.Printf("AI model:        %s\n", cfg.AI.Model)
		fmt.Println("\nReview modes:")
		fmt.Printf("  Security:      %v\n", cfg.Review.Modes.Security)
		fmt.Printf("  Performance:   %v\n", cfg.Review.Modes.Performance)
		fmt.Printf("  Style:         %v\n", cfg.Review.Modes.Style)
		fmt.Printf("  Errors:        %v\n", cfg.Review.Modes.Errors)
		fmt.Printf("  Testing:       %v\n", cfg.Review.Modes.Testing)
		fmt.Printf("  Docs:          %v\n", cfg.Review.Modes.Docs)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.GetConfigPath()
		if path == "" {
			fmt.Println("No config file found. Create one at:")
			fmt.Println("  ~/.revi.yaml (global)")
			fmt.Println("  ./.revi.yaml (project)")
		} else {
			fmt.Printf("Config file: %s\n", path)
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
}
