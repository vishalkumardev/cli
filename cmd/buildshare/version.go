package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Set at build time via ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version information",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.JSON {
			printer.JSON(map[string]string{
				"version":   Version,
				"commit":    Commit,
				"buildDate": BuildDate,
				"os":        runtime.GOOS,
				"arch":      runtime.GOARCH,
			})
			return
		}
		fmt.Printf("BuildShare CLI %s\n", Version)
		fmt.Printf("Commit:     %s\n", Commit)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Printf("OS:         %s\n", runtime.GOOS)
		fmt.Printf("Arch:       %s\n", runtime.GOARCH)
	},
}

func init() {
	if Version == "dev" || Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("BuildShare CLI {{.Version}}\n")
	rootCmd.Flags().BoolP("version", "V", false, "Print version information")
	rootCmd.AddCommand(versionCmd)
}
