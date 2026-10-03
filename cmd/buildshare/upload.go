package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildshare/cli/internal/api"
	"github.com/buildshare/cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	uploadChangelog  string
	uploadNotes      string
	uploadAppID      string
	uploadConfigFile string
)

var uploadCmd = &cobra.Command{
	Use:   "upload [platform|file]",
	Short: "Upload a build (Android APK / iOS IPA) to BuildShare",
	Long: `Upload a mobile build to BuildShare by platform (android/ios) or direct file path.

When specifying a platform, app details are read from a project configuration file (.json or .yaml):
  buildshare upload android
  buildshare upload ios
  buildshare upload android --notes "Sprint 42 test build"
  buildshare upload android --config ./buildshare.json

Direct file upload (requires project ID):
  buildshare upload my-app.apk --app <projectId>
  buildshare upload my-app.apk --app <projectId> --notes "Fixed login bug"`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetArg string
		if len(args) == 1 {
			targetArg = strings.TrimSpace(args[0])
		} else if len(args) > 1 {
			// Multiple arguments passed (e.g. unquoted glob or extra parameters)
			// Search for an argument that is a build artifact (.apk, .ipa, .aab) or platform
			for _, arg := range args {
				lower := strings.ToLower(arg)
				if lower == "android" || lower == "ios" ||
					strings.HasSuffix(lower, ".apk") ||
					strings.HasSuffix(lower, ".ipa") ||
					strings.HasSuffix(lower, ".aab") {
					targetArg = arg
					break
				}
			}
			if targetArg == "" {
				targetArg = strings.TrimSpace(args[0])
			}
		}

		lowerArg := strings.ToLower(targetArg)
		ctx := context.Background()

		var filePath string
		var appID string
		var projectCfg *config.ProjectConfig
		var configFilePath string

		if uploadAppID == "***" {
			return fmt.Errorf("invalid app ID '***'. This appears to be a secret masked in GitHub Actions logs. Please pass your actual project ID or set BUILDSHARE_APP_ID")
		}
		if uploadAppID == "" {
			if envApp := os.Getenv("BUILDSHARE_APP_ID"); envApp != "" {
				uploadAppID = envApp
			} else if envApp := os.Getenv("BUILDSHARE_PROJECT_ID"); envApp != "" {
				uploadAppID = envApp
			}
		}

		if lowerArg == "android" || lowerArg == "ios" {
			platform := lowerArg
			var err error
			projectCfg, configFilePath, err = config.LoadProjectConfig(uploadConfigFile)
			if err == nil && projectCfg != nil {
				printer.Info(fmt.Sprintf("Loaded config from %s", configFilePath))

				var configuredPath string
				if platform == "android" {
					configuredPath = projectCfg.GetAndroidPath()
				} else {
					configuredPath = projectCfg.GetIOSPath()
				}

				if configuredPath != "" {
					resolved, rerr := config.ResolveBuildArtifact(platform, configuredPath, filepath.Dir(configFilePath))
					if rerr == nil {
						filePath = resolved
					}
				}

				appID = projectCfg.GetProjectID()
			}

			// If filePath was not resolved from config file, auto-discover in standard paths
			if filePath == "" {
				resolved, rerr := config.ResolveArtifactFromFileOrDir("", platform)
				if rerr != nil {
					if err != nil {
						return fmt.Errorf("could not locate %s build artifact and no valid config found: %w", platform, err)
					}
					return fmt.Errorf("no %s build artifact found in standard output paths: %w", platform, rerr)
				}
				filePath = resolved
				printer.Info(fmt.Sprintf("Discovered %s build artifact at: %s", platform, filePath))
			}
		} else if targetArg != "" {
			// Direct file / directory / pattern upload mode
			resolved, err := config.ResolveArtifactFromFileOrDir(targetArg, "auto")
			if err != nil {
				return fmt.Errorf("invalid platform or file not found: %s (expected 'android', 'ios', or a valid file path)", targetArg)
			}
			if resolved != targetArg {
				printer.Info(fmt.Sprintf("Located build artifact: %s", resolved))
			}
			filePath = resolved

			// If app ID is not explicitly provided, try to load project config if present
			if uploadAppID == "" {
				if cfg, cfgPath, err := config.LoadProjectConfig(uploadConfigFile); err == nil && cfg != nil {
					projectCfg = cfg
					configFilePath = cfgPath
					appID = projectCfg.GetProjectID()
				}
			}
		} else {
			// No target argument provided: auto-detect from project config or filesystem
			if cfg, cfgPath, err := config.LoadProjectConfig(uploadConfigFile); err == nil && cfg != nil {
				projectCfg = cfg
				configFilePath = cfgPath
				appID = projectCfg.GetProjectID()
				printer.Info(fmt.Sprintf("Loaded config from %s", configFilePath))

				if p := projectCfg.GetAndroidPath(); p != "" {
					if resolved, err := config.ResolveBuildArtifact("android", p, filepath.Dir(configFilePath)); err == nil {
						filePath = resolved
					}
				}
				if filePath == "" {
					if p := projectCfg.GetIOSPath(); p != "" {
						if resolved, err := config.ResolveBuildArtifact("ios", p, filepath.Dir(configFilePath)); err == nil {
							filePath = resolved
						}
					}
				}
			}
			if filePath == "" {
				resolved, err := config.ResolveArtifactFromFileOrDir("", "auto")
				if err != nil {
					return fmt.Errorf("no build artifact specified and none found in standard build directories\nUsage: buildshare upload <android|ios|path-to-file> --app <projectId>")
				}
				filePath = resolved
				printer.Info(fmt.Sprintf("Auto-detected build artifact: %s", filePath))
			}
		}

		// CLI flag overrides project config
		if uploadAppID != "" {
			appID = uploadAppID
		}

		if appID == "" {
			return fmt.Errorf("project ID is required: specify 'projectId' in project config (e.g. buildshare.json), initialize with 'buildshare init', or provide '--app <projectId>'")
		}

		requireAuth()
		endpoint := fmt.Sprintf("/builds/%s/upload", appID)

		client := newClient()

		extra := map[string]string{}
		notes := uploadChangelog
		if notes == "" && uploadNotes != "" {
			notes = uploadNotes
		}
		if notes != "" {
			extra["changelog"] = notes
		}

		if projectCfg != nil {
			if name := projectCfg.GetProjectName(); name != "" {
				printer.KeyValue("Project", name)
			}
			if branch := projectCfg.GetDefaultBranch(); branch != "" {
				printer.KeyValue("Branch", branch)
			}
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
	uploadCmd.Flags().StringVar(&uploadAppID, "app", "", "App / Project ID (overrides config file)")
	uploadCmd.Flags().StringVar(&uploadAppID, "project", "", "App / Project ID (alias for --app)")
	uploadCmd.Flags().StringVar(&uploadChangelog, "changelog", "", "Release notes for this build")
	uploadCmd.Flags().StringVar(&uploadNotes, "notes", "", "Release notes for this build (alias for --changelog)")
	uploadCmd.Flags().StringVarP(&uploadConfigFile, "config", "c", "", "Path to project configuration file (.json or .yaml)")
	rootCmd.AddCommand(uploadCmd)
}
