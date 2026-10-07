package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var kidPenaltiesCmd = &cobra.Command{
	Use:   "apply-kid-penalties",
	Short: "Deduct a penalty fee for assigned kid tasks left incomplete past their due date (run on a schedule, e.g. the daily GitHub Actions workflow in .github/workflows/kid-task-penalties.yml)",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()

		applied, err := deps.Services.Wallet.RunDailyPenalties(ctx)
		if err != nil {
			return fmt.Errorf("apply-kid-penalties: %w", err)
		}
		fmt.Printf("apply-kid-penalties: penalized %d task(s)\n", applied)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(kidPenaltiesCmd)
}
