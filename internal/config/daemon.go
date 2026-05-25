package config

import (
	"github.com/kilip/sbctl/internal/daemon"
)

func (c *Config) NewDaemon() *daemon.Daemon {
	configPath := c.v.ConfigFileUsed()
	l := SetupLogger(c)

	// Provider this will be called every time reload
	provider := func() []daemon.Worker {
		cfg := GetConfig()

		return []daemon.Worker{
			cfg.GetGitSync(),
			// Add other workers here as they are implemented
		}
	}

	d := daemon.NewDaemon(provider, configPath, l)

	OnReload(func(cfg *Config) {
		d.Reload()
	})

	return d
}

func BootstrapDaemon() *daemon.Daemon {
	return GetConfig().NewDaemon()
}
