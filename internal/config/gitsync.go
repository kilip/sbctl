package config

import (
	"time"

	"github.com/kilip/sbctl/internal/daemon"
	"github.com/kilip/sbctl/internal/gitsync"
	"github.com/spf13/viper"
)

// gitsyncDefaults sets the default configuration for gitsync.
func gitsyncDefaults(v *viper.Viper) {
	v.SetDefault("gitsync.enabled", false)
	v.SetDefault("gitsync.git_repository", "")
	v.SetDefault("gitsync.debounce", 10*time.Second)
}

func (c *Config) GetGitSync() daemon.Worker {
	return gitsync.NewGitSync(&c.GitSync)
}

func (c *Config) GetGitSyncSSH() error {
	return gitsync.ConfigureSSH(c.ConfigDir, c.Vault.Dir, c.Vault.UserEmail)
}
