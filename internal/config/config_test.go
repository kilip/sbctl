package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestInitConfig_DevMode(t *testing.T) {
	viper.Reset()

	root, ok := findProjectRoot()
	if !ok {
		t.Fatal("could not find project root")
	}

	// initConfig should pick up testdata/default/config.json in dev mode
	err := initConfig("")
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	expected := filepath.Join(root, "testdata", "default", "config.json")
	if viper.ConfigFileUsed() != expected {
		t.Errorf("expected %s, got %s", expected, viper.ConfigFileUsed())
	}
}

func TestInitConfig_Env(t *testing.T) {
	// Reset viper for test
	viper.Reset()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	os.Setenv("SBCTL_LOG_LEVEL", "debug")
	defer os.Unsetenv("SBCTL_LOG_LEVEL")

	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	if GetConfig().Log.Level != LogLevelDebug {
		t.Errorf("expected LogLevel debug, got %s", GetConfig().Log.Level)
	}
}

func TestVaultConfig(t *testing.T) {
	viper.Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	os.Setenv("SBCTL_VAULT_DIR", "/tmp/vault")
	os.Setenv("SBCTL_VAULT_USER_NAME", "John Doe")
	os.Setenv("SBCTL_VAULT_USER_EMAIL", "john@example.com")
	os.Setenv("SBCTL_VAULT_GIT_REPOSITORY", "git@github.com:user/repo.git")
	defer func() {
		os.Unsetenv("SBCTL_VAULT_DIR")
		os.Unsetenv("SBCTL_VAULT_USER_NAME")
		os.Unsetenv("SBCTL_VAULT_USER_EMAIL")
		os.Unsetenv("SBCTL_VAULT_GIT_REPOSITORY")
	}()

	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	cfg := GetConfig()
	if cfg.Vault.Dir != "/tmp/vault" {
		t.Errorf("expected vault dir /tmp/vault, got %s", cfg.Vault.Dir)
	}
	if cfg.Vault.UserName != "John Doe" {
		t.Errorf("expected vault user_name John Doe, got %s", cfg.Vault.UserName)
	}
	if cfg.Vault.UserEmail != "john@example.com" {
		t.Errorf("expected vault user_email john@example.com, got %s", cfg.Vault.UserEmail)
	}
	if cfg.Vault.GitRepository != "git@github.com:user/repo.git" {
		t.Errorf("expected vault git_repository git@github.com:user/repo.git, got %s", cfg.Vault.GitRepository)
	}
}
