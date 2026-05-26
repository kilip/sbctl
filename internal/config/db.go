package config

import "github.com/spf13/viper"

type DbConfig struct {
	Path   string `mapstructure:"path" json:"path"`
	Driver string `mapstructure:"driver" json:"driver"`
}

// dbDefaults sets the default configuration for the database.
func dbDefaults(v *viper.Viper) {
	v.SetDefault("db.path", "~/.sbctl/sbctl.db")
	v.SetDefault("db.driver", "sqlite")
}
