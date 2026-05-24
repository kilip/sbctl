package config

import (
	"github.com/kilip/sbctl/internal/daemon"
	"github.com/spf13/viper"
)

func (c *Config) NewDaemon() *daemon.Daemon {
	configPath := viper.ConfigFileUsed()

	// Provider this will be called every time reload
	provider := func() []daemon.Worker {
		// Re-read config from file to get latest values
		_ = initConfig(configPath)
		cfg := GetConfig()

		return []daemon.Worker{
			cfg.GetGitSync(),
			// Add other workers here as they are implemented
		}
	}

	return daemon.NewDaemon(provider, configPath)
}

func BootstrapDaemon() *daemon.Daemon {
	return GetConfig().NewDaemon()
}
