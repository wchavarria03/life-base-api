package cmd

import "github.com/spf13/cobra"

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump raw extracted text from PDFs (for debugging parsers)",
	RunE: func(_ *cobra.Command, args []string) error {
		return deps.Handlers.Dump.Handle(cfg.InputDir, cfg.OutputDir, cfg.Verbose)
	},
}

func init() {
	rootCmd.AddCommand(dumpCmd)
}
