package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"life-base-api/app/internal/core"
)

type cliConfig struct {
	Verbose             bool
	OutputDir           string
	InputDir            string
	DryRun              bool
	SupabaseURL         string
	SupabaseKey         string
	SupabaseAnonKey     string
	ServerAddr          string
	UserID              string
	AllowedOrigins      string
	MetaAccessToken     string
	MetaFacebookPageID  string
	MetaInstagramUserID string
	StravaClientID      string
	StravaClientSecret  string
	VAPIDPublicKey      string
	VAPIDPrivateKey     string
	VAPIDSubject        string
	ResendAPIKey        string
	ResendFrom          string
}

var (
	cfg  = &cliConfig{}
	deps *core.Dependencies
)

var rootCmd = &cobra.Command{
	Use:   "life-base-api",
	Short: "Ingest bank statement PDFs, store transactions, and serve them via HTTP API",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(cfg.InputDir); os.IsNotExist(err) {
			return fmt.Errorf("input directory does not exist: %s", cfg.InputDir)
		}
		if err := os.MkdirAll(cfg.OutputDir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		origins := []string{"*"}
		if cfg.AllowedOrigins != "" {
			origins = strings.Split(cfg.AllowedOrigins, ",")
		}

		var err error
		deps, err = core.NewDependencies(core.Config{
			SupabaseURL:         cfg.SupabaseURL,
			SupabaseKey:         cfg.SupabaseKey,
			SupabaseAnonKey:     cfg.SupabaseAnonKey,
			ServerAddr:          cfg.ServerAddr,
			UserID:              cfg.UserID,
			AllowedOrigins:      origins,
			MetaAccessToken:     cfg.MetaAccessToken,
			MetaFacebookPageID:  cfg.MetaFacebookPageID,
			MetaInstagramUserID: cfg.MetaInstagramUserID,
			StravaClientID:      cfg.StravaClientID,
			StravaClientSecret:  cfg.StravaClientSecret,
			VAPIDPublicKey:      cfg.VAPIDPublicKey,
			VAPIDPrivateKey:     cfg.VAPIDPrivateKey,
			VAPIDSubject:        cfg.VAPIDSubject,
			ResendAPIKey:        cfg.ResendAPIKey,
			ResendFrom:          cfg.ResendFrom,
		})
		if err != nil {
			return fmt.Errorf("initialising dependencies: %w", err)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&cfg.OutputDir, "output", "o", "data/output", "Output directory")
	rootCmd.PersistentFlags().StringVarP(&cfg.InputDir, "input-dir", "i", "data/input", "Input PDF directory")
	rootCmd.PersistentFlags().BoolVar(&cfg.DryRun, "dry-run", false, "Write to files instead of Supabase")
	rootCmd.PersistentFlags().StringVar(&cfg.SupabaseURL, "supabase-url", os.Getenv("SUPABASE_URL"), "Supabase project URL")
	rootCmd.PersistentFlags().StringVar(&cfg.SupabaseKey, "supabase-key", os.Getenv("SUPABASE_KEY"), "Supabase service role key")
	rootCmd.PersistentFlags().StringVar(&cfg.SupabaseAnonKey, "supabase-anon-key", os.Getenv("SUPABASE_ANON_KEY"), "Supabase anon key (for user-context requests)")
	defaultAddr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		defaultAddr = ":" + port
	}
	rootCmd.PersistentFlags().StringVar(&cfg.ServerAddr, "addr", defaultAddr, "HTTP server listen address")
	rootCmd.PersistentFlags().StringVar(&cfg.UserID, "user-id", os.Getenv("LEDGER_USER_ID"), "Supabase user ID to associate imported data with")
	rootCmd.PersistentFlags().StringVar(&cfg.AllowedOrigins, "cors-origins", os.Getenv("ALLOWED_ORIGINS"), "Comma-separated allowed CORS origins (default: *)")
	rootCmd.PersistentFlags().StringVar(&cfg.MetaAccessToken, "meta-access-token", os.Getenv("META_ACCESS_TOKEN"), "Meta (Facebook/Instagram) page access token")
	rootCmd.PersistentFlags().StringVar(&cfg.MetaFacebookPageID, "meta-facebook-page-id", os.Getenv("META_FACEBOOK_PAGE_ID"), "Facebook page ID to post to")
	rootCmd.PersistentFlags().StringVar(&cfg.MetaInstagramUserID, "meta-instagram-user-id", os.Getenv("META_INSTAGRAM_USER_ID"), "Instagram business user ID to post to")
	rootCmd.PersistentFlags().StringVar(&cfg.StravaClientID, "strava-client-id", os.Getenv("STRAVA_CLIENT_ID"), "Strava API client ID")
	rootCmd.PersistentFlags().StringVar(&cfg.StravaClientSecret, "strava-client-secret", os.Getenv("STRAVA_CLIENT_SECRET"), "Strava API client secret")
	rootCmd.PersistentFlags().StringVar(&cfg.VAPIDPublicKey, "vapid-public-key", os.Getenv("VAPID_PUBLIC_KEY"), "VAPID public key for web push")
	rootCmd.PersistentFlags().StringVar(&cfg.VAPIDPrivateKey, "vapid-private-key", os.Getenv("VAPID_PRIVATE_KEY"), "VAPID private key for web push")
	rootCmd.PersistentFlags().StringVar(&cfg.VAPIDSubject, "vapid-subject", os.Getenv("VAPID_SUBJECT"), "VAPID subject (mailto: contact) for web push")
	rootCmd.PersistentFlags().StringVar(&cfg.ResendAPIKey, "resend-api-key", os.Getenv("RESEND_API_KEY"), "Resend API key for the email digest")
	rootCmd.PersistentFlags().StringVar(&cfg.ResendFrom, "resend-from", os.Getenv("RESEND_FROM"), "From address for the email digest, e.g. \"Life-Base <notifications@yourdomain.com>\"")
}
