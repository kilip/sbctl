package config

import (
	"github.com/kilip/sbctl/internal/shared/logger"
	"github.com/spf13/viper"
)

type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogConfig struct {
	Level   LogLevel `mapstructure:"level" json:"level"`
	Adapter string   `mapstructure:"adapter" json:"adapter"`
}

// loggerDefaults sets the default configuration for the logger.
func loggerDefaults(v *viper.Viper) {
	v.SetDefault("log.level", "info")
	v.SetDefault("log.adapter", "slog")
}

// SetupLogger initializes the logger based on the application configuration.
func SetupLogger(cfg *Config) logger.Logger {
	switch cfg.Log.Adapter {
	case "slog":
		fallthrough
	default:
		return logger.NewSlogLogger(string(cfg.Log.Level))
	}
}
