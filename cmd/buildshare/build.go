package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buildshare/cli/internal/api"
	"github.com/buildshare/cli/internal/config"
	"github.com/spf13/cobra"
)

// ── Parent command ────────────────────────────────────────────────────────────

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Manage builds",
	Long:  "List builds, check status, and get build details.",
}

// ── build list ────────────────────────────────────────────────────────────────

var (
	buildListAppID      string
	buildListProjectID  string
	buildListConfigFile string
)

var buildListCmd = &cobra.Command{
	Use:   "list",
	Short: "List builds for an app or project",
	Long: `List builds for a project.

If --app or --project is not provided, you will be prompted to select from your projects:
  buildshare build list
  buildshare build list --app <projectId>
  buildshare build list --project <projectId>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		ctx := context.Background()
		client := newClient()

		targetAppID := buildListAppID
		if targetAppID == "" && buildListProjectID != "" {
			targetAppID = buildListProjectID
		}

		var targetAppName string

		// If --app / --project is not provided, check local project config first
		if targetAppID == "" {
			if projCfg, _, err := config.LoadProjectConfig(buildListConfigFile); err == nil && projCfg != nil {
				targetAppID = projCfg.GetProjectID()
				targetAppName = projCfg.GetProjectName()
			}
		}

		// If still empty, fetch and display all projects to select from
		if targetAppID == "" {
			resp, err := client.Post(ctx, "/apps/list", api.PaginationRequest{Page: 1, PageSize: 50})
			if err != nil {
				return fmt.Errorf("failed to fetch projects: %w", err)
			}

			var appList api.AppListResponse
			if err := api.Decode(resp.Data, &appList); err != nil {
				return err
			}

			items := appList.Items()
			if len(items) == 0 {
				printer.Info("No projects found. Create one with: buildshare app create")
				return nil
			}

			if cfg.CI {
				return fmt.Errorf("--app flag is required in CI mode")
			}

			printer.Info("Select a project:")
			for i, app := range items {
				fmt.Printf("  %d) %s (%s)\n", i+1, app.Name, app.PackageName)
			}
			printer.Newline()

			fmt.Print("  Select an option [1]: ")
			reader := bufio.NewReader(os.Stdin)
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			selectedIdx := 0
			if choice == "" {
				selectedIdx = 0
			} else {
				// Check if user entered an App ID directly
				for _, app := range items {
					if strings.EqualFold(choice, app.AppID) {
						targetAppID = app.AppID
						targetAppName = app.Name
						break
					}
				}

				if targetAppID == "" {
					fmt.Sscanf(choice, "%d", &selectedIdx)
					selectedIdx-- // convert 1-based to 0-based
				}
			}

			if targetAppID == "" {
				if selectedIdx < 0 || selectedIdx >= len(items) {
					printer.Error("Invalid option selected.")
					return nil
				}
				targetAppID = items[selectedIdx].AppID
				targetAppName = items[selectedIdx].Name
			}
		}

		resp, err := client.Post(ctx, fmt.Sprintf("/builds/%s/list", targetAppID), api.PaginationRequest{Page: 1, PageSize: 20})
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
			if targetAppName != "" {
				printer.Info(fmt.Sprintf("No builds found for %s.", targetAppName))
			} else {
				printer.Info("No builds found.")
			}
			return nil
		}

		total := list.Total
		if total == 0 {
			total = len(items)
		}

		headerTitle := fmt.Sprintf("Builds (%d)", total)
		if targetAppName != "" {
			headerTitle = fmt.Sprintf("Builds for %s (%d)", targetAppName, total)
		}
		printer.Header(headerTitle)

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
	buildListCmd.Flags().StringVar(&buildListAppID, "app", "", "App / Project ID to list builds for")
	buildListCmd.Flags().StringVar(&buildListProjectID, "project", "", "App / Project ID to list builds for (alias for --app)")
	buildListCmd.Flags().StringVarP(&buildListConfigFile, "config", "c", "", "Path to project configuration file (.json or .yaml)")

	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildInfoCmd)
	rootCmd.AddCommand(buildCmd)
}
