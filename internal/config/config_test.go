package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestInitConfig_DevMode(t *testing.T) {
	viper.Reset()

	testCwd, _ := os.Getwd()
	_, errStat := os.Stat("go.mod")
	t.Logf("CWD: %s, go.mod err: %v", testCwd, errStat)

	// initConfig should pick up .sbctl/config.json in CWD because go.mod exists
	err := initConfig("")
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	defer os.RemoveAll(filepath.Join(testCwd, ".sbctl"))

	expected := filepath.Join(testCwd, ".sbctl", "config.json")
	if viper.ConfigFileUsed() != expected {
		t.Errorf("expected %s, got %s", expected, viper.ConfigFileUsed())
	}
}

func TestInitConfig_Env(t *testing.T) {
	// Reset viper for test
	viper.Reset()

	testCwd, _ := os.Getwd()
	defer os.RemoveAll(filepath.Join(testCwd, ".sbctl"))

	os.Setenv("SBCTL_LOG_LEVEL", "debug")
	defer os.Unsetenv("SBCTL_LOG_LEVEL")

	err := initConfig("")
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	if GetConfig().Log.Level != LogLevelDebug {
		t.Errorf("expected LogLevel debug, got %s", GetConfig().Log.Level)
	}
}

func TestVaultConfig(t *testing.T) {
	viper.Reset()
	testCwd, _ := os.Getwd()
	defer os.RemoveAll(filepath.Join(testCwd, ".sbctl"))

	os.Setenv("SBCTL_VAULT_DIR", "/tmp/vault")
	defer os.Unsetenv("SBCTL_VAULT_DIR")

	err := initConfig("")
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	cfg := GetConfig()
	if cfg.Vault.Dir != "/tmp/vault" {
		t.Errorf("expected vault dir /tmp/vault, got %s", cfg.Vault.Dir)
	}
}
