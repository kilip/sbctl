package config

import (
	"github.com/kilip/sbctl/internal/shared/logger"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogConfig struct {
	Level   LogLevel `mapstructure:"level"`
	Adapter string   `mapstructure:"adapter"`
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
