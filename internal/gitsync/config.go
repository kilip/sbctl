package gitsync

import "time"

type Config struct {
	Dir           string        `mapstructure:"dir"`
	UserName      string        `mapstructure:"user_name"`
	UserEmail     string        `mapstructure:"user_email"`
	GitRepository string        `mapstructure:"git_repository"`
	Enabled       bool          `mapstructure:"enabled"`
	Debounce      time.Duration `mapstructure:"debounce"`
}
