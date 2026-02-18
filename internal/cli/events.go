package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/youmna-rabie/claude-pod/internal/event"
)

var eventsLimit int

func init() {
	listEventsCmd.Flags().IntVar(&eventsLimit, "limit", 20, "maximum number of events to display")
	rootCmd.AddCommand(listEventsCmd)
}

var listEventsCmd = &cobra.Command{
	Use:   "list-events",
	Short: "Print recent events from store",
	RunE:  listEvents,
}

func listEvents(cmd *cobra.Command, args []string) error {
	// Events are in-memory only, so we create a fresh store and show it's empty.
	// A running gateway would need an API call; for now we demonstrate the store query.
	store, err := event.NewMemoryStore(1000)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	events, err := store.List(eventsLimit, 0)
	if err != nil {
		return fmt.Errorf("listing events: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No events found. (Start the gateway to receive events.)")
		return nil
	}

	fmt.Printf("%-36s  %-15s  %-12s  %s\n", "ID", "CHANNEL", "STATUS", "TIMESTAMP")
	for _, e := range events {
		fmt.Printf("%-36s  %-15s  %-12s  %s\n", e.ID, e.ChannelID, e.Status, e.Timestamp.Format("2006-01-02 15:04:05"))
	}
	return nil
}
