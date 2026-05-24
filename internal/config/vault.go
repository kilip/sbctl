package config

import "github.com/spf13/viper"

type VaultConfig struct {
	Dir           string `mapstructure:"dir"`
	UserName      string `mapstructure:"user_name"`
	UserEmail     string `mapstructure:"user_email"`
	GitRepository string `mapstructure:"git_repository"`
}

// vaultDefaults sets the default configuration for the vault.
func vaultDefaults() {
	viper.SetDefault("vault.dir", "")
	viper.SetDefault("vault.user_name", "")
	viper.SetDefault("vault.user_email", "")
	viper.SetDefault("vault.git_repository", "")
}
