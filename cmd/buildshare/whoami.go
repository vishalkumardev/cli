package cmd

import (
	"context"

	"github.com/buildshare/cli/internal/api"
	"github.com/buildshare/cli/internal/auth"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		ctx := context.Background()
		client := newClient()

		resp, err := client.Get(ctx, "/user/profile")
		if err != nil {
			if api.IsUnauthorized(err) {
				printer.Error("Session expired. Please run: buildshare login")
				return nil
			}
			return err
		}

		var profile api.UserProfile
		if err := api.Decode(resp.Data, &profile); err != nil {
			return err
		}

		if cfg.JSON {
			printer.JSON(profile)
			return nil
		}

		creds := auth.Load()
		source := "stored credentials"
		if tokenFlag != "" {
			source = "--token flag"
		} else if t := auth.ResolveToken(); creds == nil || t != creds.Token {
			source = "BUILDSHARE_TOKEN env"
		}

		printer.Header("Authenticated User")
		printer.KeyValue("Name", profile.Name)
		printer.KeyValue("Email", profile.Email)
		printer.KeyValue("Auth Source", source)
		printer.Newline()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
