package cmd

import (
	"github.com/spf13/cobra"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// adminCmd groups admin related cli commands together
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin management commands",
	Long:  "Commands for managing admin users and system administration.",
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// init adds the admin command to the root command
func init() {
	rootCmd.AddCommand(adminCmd)
}
