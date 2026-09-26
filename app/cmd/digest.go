package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
	Use:   "send-digest",
	Short: "Send push/email notifications to users with overdue reminders (run on a schedule, e.g. the daily GitHub Actions workflow in .github/workflows/digest.yml)",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()

		pushSent, err := deps.Services.Digest.RunPush(ctx)
		if err != nil {
			fmt.Printf("send-digest: push skipped: %v\n", err)
		} else {
			fmt.Printf("send-digest: sent %d push notification(s)\n", pushSent)
		}

		emailSent, err := deps.Services.Digest.RunEmail(ctx)
		if err != nil {
			fmt.Printf("send-digest: email skipped: %v\n", err)
		} else {
			fmt.Printf("send-digest: sent %d email(s)\n", emailSent)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(digestCmd)
}
