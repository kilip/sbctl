package config

import (
	"time"

	"github.com/spf13/viper"
)

type GitSyncConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Remote   string        `mapstructure:"remote"`
	Debounce time.Duration `mapstructure:"debounce"`
}

// gitsyncDefaults sets the default configuration for gitsync.
func gitsyncDefaults() {
	viper.SetDefault("gitsync.enabled", false)
	viper.SetDefault("gitsync.remote", "")
	viper.SetDefault("gitsync.debounce", 10*time.Second)
}
