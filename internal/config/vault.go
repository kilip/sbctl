package config

import "github.com/spf13/viper"

type VaultConfig struct {
	Dir string `mapstructure:"dir"`
}

// vaultDefaults sets the default configuration for the vault.
func vaultDefaults() {
	viper.SetDefault("vault.dir", "")
}
