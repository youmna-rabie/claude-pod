package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "gateway",
	Short: "claude-pod event gateway",
	Long:  "claude-pod gateway receives webhooks, routes events to an agent, and exposes admin endpoints.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "gateway.yaml", "path to configuration file")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
