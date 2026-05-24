package main

import (
	"github.com/kilip/sbctl/internal/config"
	"github.com/spf13/cobra"
)

var syncSshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Auto-configure SSH key for GitHub synchronization",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.GetConfig()
		return cfg.GetGitSyncSSH()
	},
}

func init() {
	syncCmd.AddCommand(syncSshCmd)
}
