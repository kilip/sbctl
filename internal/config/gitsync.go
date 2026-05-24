package config

import (
	"time"

	"github.com/spf13/viper"
)

// gitsyncDefaults sets the default configuration for gitsync.
func gitsyncDefaults() {
	viper.SetDefault("gitsync.enabled", false)
	viper.SetDefault("gitsync.git_repository", "")
	viper.SetDefault("gitsync.debounce", 10*time.Second)
}
