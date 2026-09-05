package cmd

import (
	"context"
	"fmt"

	"github.com/buildshare/cli/internal/api"
	"github.com/spf13/cobra"
)

var uploadChangelog string
var uploadAppID string

var uploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload a build (APK/IPA) to BuildShare",
	Long: `Upload an APK or IPA file to BuildShare.

Public upload (no auth required, expires in 7 days):
  buildshare upload my-app.apk

Private upload (requires auth + app ID):
  buildshare upload my-app.apk --app <appId>
  buildshare upload my-app.apk --app <appId> --changelog "Fixed login bug"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		ctx := context.Background()

		var endpoint string
		if uploadAppID != "" {
			requireAuth()
			endpoint = fmt.Sprintf("/builds/%s/upload", uploadAppID)
		} else {
			endpoint = "/builds/upload"
		}

		client := newClient()

		extra := map[string]string{}
		if uploadChangelog != "" {
			extra["changelog"] = uploadChangelog
		}

		printer.Info("Uploading " + filePath + "...")
		printer.Newline()

		resp, err := client.UploadFile(ctx, endpoint, filePath, "apk", extra, func(pct int) {
			printer.Progress(pct, "Uploading...")
		})
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}

		var build api.Build
		if err := api.Decode(resp.Data, &build); err != nil {
			return err
		}

		if cfg.JSON {
			printer.JSON(build)
			return nil
		}

		printer.Newline()
		printer.Success("Build uploaded successfully!")
		printer.Newline()
		printer.KeyValue("Build ID", build.BuildID)
		printer.KeyValue("App Name", build.AppName)
		printer.KeyValue("Version", fmt.Sprintf("%s (%d)", build.VersionName, build.VersionCode))
		printer.KeyValue("Platform", build.Platform)
		printer.KeyValue("Size", fmt.Sprintf("%.2f MB", build.FileSize))

		if build.InstallURL != "" {
			printer.KeyValue("Install URL", build.InstallURL)
		}

		shareURL := fmt.Sprintf("https://buildshare.in/build/%s", build.BuildID)
		printer.KeyValue("Share URL", shareURL)
		printer.Newline()

		return nil
	},
}

func init() {
	uploadCmd.Flags().StringVar(&uploadAppID, "app", "", "App ID for private upload (requires auth)")
	uploadCmd.Flags().StringVar(&uploadChangelog, "changelog", "", "Release notes for this build")
	rootCmd.AddCommand(uploadCmd)
}
