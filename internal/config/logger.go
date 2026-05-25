package config

import (
	"github.com/kilip/sbctl/internal/shared/logger"
	"github.com/spf13/viper"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogConfig struct {
	Level   LogLevel `mapstructure:"level" json:"level"`
	Adapter string   `mapstructure:"adapter" json:"adapter"`
}

// loggerDefaults sets the default configuration for the logger.
func loggerDefaults() {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.adapter", "slog")
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
