package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	RunE: func(_ *cobra.Command, _ []string) error {
		go func() {
			if err := deps.Server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			}
		}()

		fmt.Printf("API server listening on %s\n", cfg.ServerAddr)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return deps.Server.Stop(ctx)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
