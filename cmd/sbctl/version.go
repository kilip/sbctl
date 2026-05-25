package main

import (
	"fmt"

	"github.com/kilip/sbctl/internal/core"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of sbctl",
	Long:  `All software has versions. This is sbctl's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("sbctl version %s, commit %s, built at %s\n", core.Version, core.Commit, core.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
