package main

import (
	"fmt"
	"os"

	"github.com/kilip/sbctl/internal/core"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade sbctl to the latest version",
	Long:  `Downloads and installs the latest version of sbctl from GitHub Releases.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := core.Upgrade(); err != nil {
			fmt.Fprintf(os.Stderr, "Error upgrading: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
