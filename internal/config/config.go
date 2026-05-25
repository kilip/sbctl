package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kilip/sbctl/internal/core"
	"github.com/kilip/sbctl/internal/gitsync"
	"github.com/spf13/viper"
)

type Config struct {
	ConfigDir string         `mapstructure:"-" json:"-"`
	Log       LogConfig      `mapstructure:"log" json:"log"`
	Vault     VaultConfig    `mapstructure:"vault" json:"vault"`
	GitSync   gitsync.Config `mapstructure:"gitsync" json:"gitsync"`
}

var (
	instance *Config
	once     sync.Once
)

const SchemaURL = "https://raw.githubusercontent.com/kilip/sbctl/main/docs/config-schema.json"

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
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			// Basic check if it's our module to avoid hijacking in other projects
			content, err := os.ReadFile(modPath)
			if err == nil && strings.Contains(string(content), "module github.com/kilip/sbctl") {
				return dir, true
			}
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
		// Auto-detect dev mode: if version is 'dev' and go.mod exists in current or parent dirs,
		// use testdata/default/config.json.
		if core.Version == "dev" {
			if root, ok := findProjectRoot(); ok {
				configPath = filepath.Join(root, "testdata", "default", "config.json")
			}
		}

		if configPath == "" {
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

	_ = instance.ensureSchema()

	return nil
}

func (c *Config) ensureSchema() error {
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		return nil
	}

	// Read file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(content, &m); err != nil {
		// Not a valid JSON or empty, we skip auto-injection to avoid corruption
		return nil
	}

	if _, ok := m["$schema"]; ok {
		return nil
	}

	// Inject $schema
	m["$schema"] = SchemaURL

	newContent, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newContent, 0644)
}

// Save writes the current configuration back to the config file.
func (c *Config) Save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("error unmarshaling config to map: %w", err)
	}

	m["$schema"] = SchemaURL

	for k, v := range m {
		viper.Set(k, v)
	}

	return viper.WriteConfig()
}
