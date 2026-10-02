package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var houseTimersCmd = &cobra.Command{
	Use:   "notify-house-timers",
	Short: "Push-notify owners of any fired house timers (run on a short schedule, e.g. every 5 min via GitHub Actions)",
	RunE: func(_ *cobra.Command, _ []string) error {
		sent, err := deps.Services.HouseTimer.NotifyDue(context.Background())
		if err != nil {
			return err
		}
		fmt.Printf("notify-house-timers: sent %d\n", sent)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(houseTimersCmd)
}
