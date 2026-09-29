package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/buildshare/cli/internal/api"
	"github.com/buildshare/cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	initAndroidPath string
	initIOSPath     string
	initBranch      string
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
				return configureAndSaveProject(reader, selected.AppID, selected.Name)
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

		printer.Newline()
		printer.Success(fmt.Sprintf("Created app: %s", app.Name))
		printer.KeyValue("App ID", app.AppID)

		return configureAndSaveProject(reader, app.AppID, app.Name)
	},
}

func configureAndSaveProject(reader *bufio.Reader, appID, appName string) error {
	defaultAndroid := detectAndroidDir()
	defaultIOS := detectIOSDir()
	defaultBranch := detectGitBranch()

	var androidPath string
	if initAndroidPath != "" {
		androidPath = initAndroidPath
	} else if cfg.CI {
		androidPath = defaultAndroid
	} else {
		printer.Newline()
		printer.Info("Configure build output folders:")
		fmt.Printf("  Android build folder [%s]: ", defaultAndroid)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			androidPath = input
		} else {
			androidPath = defaultAndroid
		}
	}

	var iosPath string
	if initIOSPath != "" {
		iosPath = initIOSPath
	} else if cfg.CI {
		iosPath = defaultIOS
	} else {
		fmt.Printf("  iOS build folder [%s]: ", defaultIOS)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			iosPath = input
		} else {
			iosPath = defaultIOS
		}
	}

	var branch string
	if initBranch != "" {
		branch = initBranch
	} else if cfg.CI {
		branch = defaultBranch
	} else {
		fmt.Printf("  Default branch [%s]: ", defaultBranch)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			branch = input
		} else {
			branch = defaultBranch
		}
	}

	projCfg := &config.ProjectConfig{
		ProjectID:     appID,
		ProjectName:   appName,
		AndroidPath:   androidPath,
		IOSPath:       iosPath,
		DefaultBranch: branch,
	}

	targetFile := config.DetectTargetConfigFile()
	if err := config.SaveProjectConfig(projCfg, targetFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	printer.Newline()
	printer.Success(fmt.Sprintf("Linked to %s (%s)", appName, appID))
	printer.KeyValue("Android Path", androidPath)
	printer.KeyValue("iOS Path", iosPath)
	printer.KeyValue("Default Branch", branch)
	printer.Info(fmt.Sprintf("Config written to %s", targetFile))
	return nil
}

func detectAndroidDir() string {
	candidates := []string{
		"android/app/build/outputs/apk/release",
		"android/app/build/outputs/apk",
		"build/app/outputs/flutter-apk",
		"android",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return "./" + c
		}
	}
	return "./android/app/build/outputs/apk/release"
}

func detectIOSDir() string {
	candidates := []string{
		"ios/builds",
		"ios/build/Build/Products/Release-iphoneos",
		"build/ios/ipa",
		"ios",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return "./" + c
		}
	}
	return "./ios/builds"
}

func detectGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		b := strings.TrimSpace(string(out))
		if b != "" && b != "HEAD" {
			return b
		}
	}
	return "main"
}

func init() {
	initCmd.Flags().StringVar(&initAndroidPath, "android", "", "Android build folder path")
	initCmd.Flags().StringVar(&initIOSPath, "ios", "", "iOS build folder path")
	initCmd.Flags().StringVar(&initBranch, "branch", "", "Default git branch")
	rootCmd.AddCommand(initCmd)
}
