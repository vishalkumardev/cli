package cmd

import (
	"fmt"
	"os"

	"github.com/buildshare/cli/internal/api"
	"github.com/buildshare/cli/internal/auth"
	"github.com/buildshare/cli/internal/config"
	"github.com/buildshare/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	ciMode     bool
	verbose    bool
	cfg        *config.Config
	printer    *output.Printer
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "buildshare",
	Short: "BuildShare CLI — ship builds to your team",
	Long: `BuildShare CLI lets you upload, manage, and distribute
mobile app builds (APK/IPA) to your team from the terminal.

Get started:
  buildshare login
  buildshare upload android`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg = config.Load()
		cfg.JSON = cfg.JSON || jsonOutput
		cfg.CI = cfg.CI || ciMode
		cfg.Debug = cfg.Debug || verbose
		printer = output.New(cfg.JSON, cfg.CI)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Printf("BuildShare CLI %s\n", Version)
			return
		}
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "CI mode (no colors, no prompts)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug output")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		p := output.New(jsonOutput, ciMode)
		p.Error(err.Error())
		os.Exit(1)
	}
}

// newClient creates an authenticated API client.
func newClient() *api.Client {
	token := auth.ResolveToken()
	return api.New(config.APIURL, token)
}

// requireAuth checks that the user is logged in and exits if not.
func requireAuth() {
	if !auth.IsLoggedIn() {
		p := output.New(jsonOutput, ciMode)
		p.Error("Not logged in.")
		fmt.Fprintln(os.Stderr, "\nRun:\n\n    buildshare login\n\nor provide a token:\n\n    BUILDSHARE_TOKEN=<token> buildshare <command>")
		os.Exit(2)
	}
}
