package config

import "github.com/spf13/viper"

type VaultConfig struct {
	Dir           string `mapstructure:"dir" json:"dir"`
	UserName      string `mapstructure:"user_name" json:"user_name"`
	UserEmail     string `mapstructure:"user_email" json:"user_email"`
	GitRepository string `mapstructure:"git_repository" json:"git_repository"`
}

// vaultDefaults sets the default configuration for the vault.
func vaultDefaults(v *viper.Viper) {
	v.SetDefault("vault.dir", "")
	v.SetDefault("vault.user_name", "")
	v.SetDefault("vault.user_email", "")
	v.SetDefault("vault.git_repository", "")
}
