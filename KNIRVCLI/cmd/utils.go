package cmd

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Version information
	Version   = "0.1.0"
	BuildTime = time.Now().Format(time.RFC3339)
	GitCommit = "development"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display detailed version information about the KNIRVCHAIN CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("KNIRVCHAIN CLI v%s\n", Version)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
