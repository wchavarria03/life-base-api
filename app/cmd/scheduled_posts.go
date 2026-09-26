package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var scheduledPostsCmd = &cobra.Command{
	Use:   "send-scheduled-posts",
	Short: "Send any due scheduled social posts (run on a schedule, e.g. an hourly GitHub Actions cron)",
	RunE: func(_ *cobra.Command, _ []string) error {
		sent, failed, err := deps.Services.ScheduledPost.ProcessDue(context.Background(), "")
		if err != nil {
			return err
		}
		fmt.Printf("send-scheduled-posts: sent %d, failed %d\n", sent, failed)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scheduledPostsCmd)
}
