package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/youmna-rabie/claude-pod/internal/config"
)

func init() {
	rootCmd.AddCommand(listChannelsCmd)
}

var listChannelsCmd = &cobra.Command{
	Use:   "list-channels",
	Short: "Print configured channels",
	RunE:  listChannels,
}

func listChannels(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Channels) == 0 {
		fmt.Println("No channels configured.")
		return nil
	}

	fmt.Printf("%-20s %-15s\n", "NAME", "TYPE")
	for _, ch := range cfg.Channels {
		fmt.Printf("%-20s %-15s\n", ch.Name, ch.Type)
	}
	return nil
}
