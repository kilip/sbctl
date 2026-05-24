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
	os.Setenv("SBCTL_VAULT_USER_NAME", "John Doe")
	os.Setenv("SBCTL_VAULT_USER_EMAIL", "john@example.com")
	os.Setenv("SBCTL_VAULT_GIT_REPOSITORY", "git@github.com:user/repo.git")
	defer func() {
		os.Unsetenv("SBCTL_VAULT_DIR")
		os.Unsetenv("SBCTL_VAULT_USER_NAME")
		os.Unsetenv("SBCTL_VAULT_USER_EMAIL")
		os.Unsetenv("SBCTL_VAULT_GIT_REPOSITORY")
	}()

	err := initConfig("")
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
