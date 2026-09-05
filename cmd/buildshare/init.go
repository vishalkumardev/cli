package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buildshare/cli/internal/api"
	"github.com/buildshare/cli/internal/auth"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a BuildShare project in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		ctx := context.Background()
		client := newClient()

		printer.Header("Initialize BuildShare Project")
		printer.Newline()

		// List existing apps
		resp, err := client.Post(ctx, "/apps/list", api.PaginationRequest{Page: 1, PageSize: 50})
		if err != nil {
			return fmt.Errorf("failed to fetch apps: %w", err)
		}

		var appList api.AppListResponse
		if err := api.Decode(resp.Data, &appList); err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)

		items := appList.Items()
		if len(items) > 0 {
			printer.Info("Your apps:")
			for i, app := range items {
				fmt.Printf("  %d) %s (%s)\n", i+1, app.Name, app.PackageName)
			}
			fmt.Printf("  %d) Create a new app\n", len(items)+1)
			printer.Newline()

			fmt.Print("  Select an option: ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			idx := 0
			fmt.Sscanf(choice, "%d", &idx)

			if idx >= 1 && idx <= len(items) {
				selected := items[idx-1]
				if err := auth.InitProjectConfig(selected.AppID, selected.Name); err != nil {
					return err
				}
				printer.Success(fmt.Sprintf("Linked to %s (%s)", selected.Name, selected.AppID))
				printer.Info("Config written to .buildshare.yaml")
				return nil
			}
		}

		// Create new app
		fmt.Print("  App name: ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)

		fmt.Print("  Package name (e.g. com.example.app): ")
		pkg, _ := reader.ReadString('\n')
		pkg = strings.TrimSpace(pkg)

		if name == "" || pkg == "" {
			printer.Error("App name and package name are required.")
			return nil
		}

		resp, err = client.Post(ctx, "/apps/create", api.CreateAppRequest{
			Name:        name,
			PackageName: pkg,
		})
		if err != nil {
			return fmt.Errorf("failed to create app: %w", err)
		}

		var app api.App
		if err := api.Decode(resp.Data, &app); err != nil {
			return err
		}

		if err := auth.InitProjectConfig(app.AppID, app.Name); err != nil {
			return err
		}

		printer.Newline()
		printer.Success(fmt.Sprintf("Created app: %s", app.Name))
		printer.KeyValue("App ID", app.AppID)
		printer.Info("Config written to .buildshare.yaml")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
