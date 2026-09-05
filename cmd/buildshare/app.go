package cmd

import (
	"context"
	"fmt"

	"github.com/buildshare/cli/internal/api"
	"github.com/spf13/cobra"
)

// ── Parent command ────────────────────────────────────────────────────────────

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage BuildShare apps",
	Long:  "Create, list, and inspect apps in your BuildShare organization.",
}

// ── app list ──────────────────────────────────────────────────────────────────

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your apps",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		ctx := context.Background()
		client := newClient()

		resp, err := client.Post(ctx, "/apps/list", api.PaginationRequest{Page: 1, PageSize: 50})
		if err != nil {
			return err
		}

		var list api.AppListResponse
		if err := api.Decode(resp.Data, &list); err != nil {
			return err
		}

		if cfg.JSON {
			printer.JSON(list)
			return nil
		}

		items := list.Items()
		if len(items) == 0 {
			printer.Info("No apps found. Create one with: buildshare app create")
			return nil
		}

		total := list.Total
		if total == 0 {
			total = len(items)
		}

		printer.Header(fmt.Sprintf("Apps (%d)", total))
		rows := make([][]string, len(items))
		for i, a := range items {
			rows[i] = []string{a.AppID, a.Name, a.PackageName}
		}
		printer.Table([]string{"ID", "Name", "Package"}, rows)
		printer.Newline()
		return nil
	},
}

// ── app create ────────────────────────────────────────────────────────────────

var appCreateName string
var appCreatePkg string

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new app",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		ctx := context.Background()
		client := newClient()

		if appCreateName == "" || appCreatePkg == "" {
			printer.Error("Both --name and --package are required.")
			return nil
		}

		resp, err := client.Post(ctx, "/apps/create", api.CreateAppRequest{
			Name:        appCreateName,
			PackageName: appCreatePkg,
		})
		if err != nil {
			return err
		}

		var app api.App
		if err := api.Decode(resp.Data, &app); err != nil {
			return err
		}

		if cfg.JSON {
			printer.JSON(app)
			return nil
		}

		printer.Success("App created!")
		printer.KeyValue("App ID", app.AppID)
		printer.KeyValue("Name", app.Name)
		printer.KeyValue("Package", app.PackageName)
		printer.Newline()
		return nil
	},
}

// ── app info ──────────────────────────────────────────────────────────────────

var appInfoCmd = &cobra.Command{
	Use:   "info <appId>",
	Short: "Get app details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		ctx := context.Background()
		client := newClient()

		resp, err := client.Post(ctx, fmt.Sprintf("/apps/details/%s", args[0]), nil)
		if err != nil {
			return err
		}

		var app api.App
		if err := api.Decode(resp.Data, &app); err != nil {
			return err
		}

		if cfg.JSON {
			printer.JSON(app)
			return nil
		}

		printer.Header("App Details")
		printer.KeyValue("App ID", app.AppID)
		printer.KeyValue("Name", app.Name)
		printer.KeyValue("Package", app.PackageName)
		printer.KeyValue("Created", app.CreatedAt)
		printer.Newline()
		return nil
	},
}

func init() {
	appCreateCmd.Flags().StringVar(&appCreateName, "name", "", "App display name")
	appCreateCmd.Flags().StringVar(&appCreatePkg, "package", "", "Package name (e.g. com.example.app)")

	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appCreateCmd)
	appCmd.AddCommand(appInfoCmd)
	rootCmd.AddCommand(appCmd)
}
