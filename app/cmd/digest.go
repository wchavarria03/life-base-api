package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
	Use:   "send-digest",
	Short: "Send push/email notifications to users with overdue reminders (run on a schedule, e.g. a Render Cron Job)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		sent, err := deps.Services.Digest.RunPush(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("send-digest: sent %d push notification(s)\n", sent)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(digestCmd)
}
