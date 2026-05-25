package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kilip/sbctl/internal/gitsync"
	"github.com/spf13/viper"
)

type Config struct {
	ConfigDir string         `mapstructure:"-"`
	Log       LogConfig      `mapstructure:"log"`
	Vault     VaultConfig    `mapstructure:"vault"`
	GitSync   gitsync.Config `mapstructure:"gitsync"`
}

var (
	instance *Config
	once     sync.Once
)

// Init initializes the configuration with the given file path.
func Init(cfgFile string) error {
	return initConfig(cfgFile)
}

// GetConfig returns the global configuration singleton.
func GetConfig() *Config {
	once.Do(func() {
		if instance == nil {
			_ = initConfig("")
		}
	})
	return instance
}

func findProjectRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func initDefaults() {
	loggerDefaults()
	vaultDefaults()
	gitsyncDefaults()
}

func initConfig(cfgFile string) error {
	var configPath string
	if cfgFile != "" {
		configPath = cfgFile
	} else {
		// Auto-detect dev mode: if go.mod exists in current or parent dirs, use testdata/default/config.json
		if root, ok := findProjectRoot(); ok {
			configPath = filepath.Join(root, "testdata", "default", "config.json")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("error getting home directory: %w", err)
			}
			configPath = filepath.Join(home, ".sbctl", "config.json")
		}
	}

	viper.SetConfigFile(configPath)

	// Create directory if not exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	viper.SetEnvPrefix("SBCTL")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	initDefaults()

	if err := viper.ReadInConfig(); err != nil {
		// If config file is not found, ignore the error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			// Check if it's a path error which contains "no such file or directory"
			if !strings.Contains(err.Error(), "no such file or directory") {
				return fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	if instance == nil {
		instance = &Config{}
	}

	if err := viper.Unmarshal(instance); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	instance.ConfigDir = filepath.Dir(configPath)

	return nil
}
