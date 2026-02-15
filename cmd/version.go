package cmd

import (
	"fmt"

	"github.com/geerew/off-course/version"
	"github.com/spf13/cobra"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// versionCmd prints version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", version.GetVersion())
		fmt.Printf("Commit:  %s\n", version.GetCommit())
	},
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// init adds the version command to the root command
func init() {
	rootCmd.AddCommand(versionCmd)
}
