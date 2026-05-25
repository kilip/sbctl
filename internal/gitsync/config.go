package gitsync

import "time"

type Config struct {
	Dir           string        `mapstructure:"dir" json:"dir"`
	UserName      string        `mapstructure:"user_name" json:"user_name"`
	UserEmail     string        `mapstructure:"user_email" json:"user_email"`
	GitRepository string        `mapstructure:"git_repository" json:"git_repository"`
	Enabled       bool          `mapstructure:"enabled" json:"enabled"`
	Debounce      time.Duration `mapstructure:"debounce" json:"debounce"`
}
