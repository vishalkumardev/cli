package cmd

import (
	"context"
	"fmt"

	"github.com/buildshare/cli/internal/api"
	"github.com/spf13/cobra"
)

// ── Parent command ────────────────────────────────────────────────────────────

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Manage builds",
	Long:  "List builds, check status, and get build details.",
}

// ── build list ────────────────────────────────────────────────────────────────

var buildListAppID string

var buildListCmd = &cobra.Command{
	Use:   "list",
	Short: "List builds for an app",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		ctx := context.Background()
		client := newClient()

		if buildListAppID == "" {
			printer.Error("--app flag is required. Usage: buildshare build list --app <appId>")
			return nil
		}

		resp, err := client.Post(ctx, fmt.Sprintf("/builds/%s/list", buildListAppID), api.PaginationRequest{Page: 1, PageSize: 20})
		if err != nil {
			return err
		}

		var list api.BuildListResponse
		if err := api.Decode(resp.Data, &list); err != nil {
			return err
		}

		if cfg.JSON {
			printer.JSON(list)
			return nil
		}

		items := list.Items()
		if len(items) == 0 {
			printer.Info("No builds found.")
			return nil
		}

		total := list.Total
		if total == 0 {
			total = len(items)
		}

		printer.Header(fmt.Sprintf("Builds (%d)", total))
		rows := make([][]string, len(items))
		for i, b := range items {
			rows[i] = []string{
				b.BuildID,
				b.VersionName,
				fmt.Sprintf("%d", b.VersionCode),
				b.Platform,
				b.CreatedAt,
			}
		}
		printer.Table([]string{"Build ID", "Version", "Code", "Platform", "Created"}, rows)
		printer.Newline()
		return nil
	},
}

// ── build info ────────────────────────────────────────────────────────────────

var buildInfoCmd = &cobra.Command{
	Use:   "info <buildId>",
	Short: "Get build details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := newClient()

		resp, err := client.Get(ctx, fmt.Sprintf("/builds/%s", args[0]))
		if err != nil {
			return err
		}

		var build api.Build
		if err := api.Decode(resp.Data, &build); err != nil {
			return err
		}

		if cfg.JSON {
			printer.JSON(build)
			return nil
		}

		printer.Header("Build Details")
		printer.KeyValue("Build ID", build.BuildID)
		printer.KeyValue("App", build.AppName)
		printer.KeyValue("Package", build.PackageName)
		printer.KeyValue("Version", fmt.Sprintf("%s (%d)", build.VersionName, build.VersionCode))
		printer.KeyValue("Platform", build.Platform)
		printer.KeyValue("Size", fmt.Sprintf("%.2f MB", build.FileSize))
		printer.KeyValue("Downloads", fmt.Sprintf("%d", build.DownloadCount))
		if build.Changelog != "" {
			printer.KeyValue("Changelog", build.Changelog)
		}
		printer.KeyValue("Created", build.CreatedAt)
		printer.Newline()
		return nil
	},
}

func init() {
	buildListCmd.Flags().StringVar(&buildListAppID, "app", "", "App ID to list builds for")

	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildInfoCmd)
	rootCmd.AddCommand(buildCmd)
}
