package config

import (
	"path/filepath"

	"github.com/kilip/sbctl/internal/daemon"
)

func (c *Config) NewDaemon() *daemon.Daemon {
	configPath := filepath.Join(c.ConfigDir, "config.json")
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
