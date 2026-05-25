package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitConfig_DevMode(t *testing.T) {
	Reset()

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
	if GetConfig().v.ConfigFileUsed() != expected {
		t.Errorf("expected %s, got %s", expected, GetConfig().v.ConfigFileUsed())
	}
}

func TestInitConfig_Env(t *testing.T) {
	// Reset viper for test
	Reset()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	_ = os.Setenv("SBCTL_LOG_LEVEL", "debug")
	defer func() { _ = os.Unsetenv("SBCTL_LOG_LEVEL") }()

	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	if GetConfig().Log.Level != LogLevelDebug {
		t.Errorf("expected LogLevel debug, got %s", GetConfig().Log.Level)
	}
}

func TestVaultConfig(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	_ = os.Setenv("SBCTL_VAULT_DIR", "/tmp/vault")
	_ = os.Setenv("SBCTL_VAULT_USER_NAME", "John Doe")
	_ = os.Setenv("SBCTL_VAULT_USER_EMAIL", "john@example.com")
	_ = os.Setenv("SBCTL_VAULT_GIT_REPOSITORY", "git@github.com:user/repo.git")
	defer func() {
		_ = os.Unsetenv("SBCTL_VAULT_DIR")
		_ = os.Unsetenv("SBCTL_VAULT_USER_NAME")
		_ = os.Unsetenv("SBCTL_VAULT_USER_EMAIL")
		_ = os.Unsetenv("SBCTL_VAULT_GIT_REPOSITORY")
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

func TestSaveConfig(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	cfg := GetConfig()
	cfg.Vault.Dir = "/tmp/myvault"
	cfg.Vault.UserName = "Pak Bos"
	cfg.Vault.UserEmail = "pakbos@example.com"
	cfg.GitSync.Enabled = true

	err = cfg.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	content, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	sContent := string(content)
	// Check for snake_case fields
	expectedFields := []string{
		"\"$schema\"",
		"\"user_name\"",
		"\"user_email\"",
		"\"git_repository\"",
		"\"enabled\"",
	}

	for _, field := range expectedFields {
		if !strings.Contains(sContent, field) {
			t.Errorf("expected field %s in config file, but not found. Content:\n%s", field, sContent)
		}
	}

	// Check that PascalCase fields are NOT present (except maybe in values, but unlikely for these)
	unexpectedFields := []string{
		"\"UserName\"",
		"\"UserEmail\"",
		"\"GitRepository\"",
		"\"Enabled\"",
	}
	for _, field := range unexpectedFields {
		if strings.Contains(sContent, field) {
			t.Errorf("unexpected PascalCase field %s found in config file. Content:\n%s", field, sContent)
		}
	}
}
