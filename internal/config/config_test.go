package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/kilip/sbctl/internal/gitsync"
	"github.com/spf13/viper"
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

func TestGetGitSync(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	cfg := GetConfig()
	worker := cfg.GetGitSync()
	if worker == nil {
		t.Error("expected GetGitSync to return non-nil worker")
	} else if worker.Name() != "gitsync" {
		t.Errorf("expected worker name 'gitsync', got %s", worker.Name())
	}
}

func TestGetGitSyncSSH(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	cfg := GetConfig()
	cfg.Vault.Dir = filepath.Join(tmpDir, "vault")
	cfg.Vault.UserEmail = "test@example.com"

	// Create mock git directory
	_ = os.MkdirAll(filepath.Join(cfg.Vault.Dir, ".git"), 0755)

	// Mock gitsync.ExecCommand
	originalExec := gitsync.ExecCommand
	defer func() { gitsync.ExecCommand = originalExec }()

	gitsync.ExecCommand = func(command string, args ...string) *exec.Cmd {
		// Just run a command that exits successfully (like true or echo)
		return exec.Command("true")
	}

	err = cfg.GetGitSyncSSH()
	if err != nil {
		t.Fatalf("GetGitSyncSSH failed: %v", err)
	}
}

func TestInit(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	err := Init(cfgPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if GetConfig() == nil {
		t.Fatal("expected non-nil config after Init")
	}
}

func TestRunWizard(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	// Mock runWizardForm to return nil (simulate successful form entry)
	originalRunForm := runWizardForm
	defer func() { runWizardForm = originalRunForm }()
	runWizardForm = func(form *huh.Form) error {
		return nil
	}

	// Capture stdout to prevent cluttering test output
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	_, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() { _ = wOut.Close() }()

	// Mock gitsync.ExecCommand so configuring SSH in wizard doesn't fail
	originalExec := gitsync.ExecCommand
	defer func() { gitsync.ExecCommand = originalExec }()
	gitsync.ExecCommand = func(command string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	// Call RunWizard
	err = RunWizard()
	if err != nil {
		t.Fatalf("RunWizard failed: %v", err)
	}
}

func TestFindProjectRoot_Fail(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	_, ok := findProjectRoot()
	if ok {
		t.Error("expected findProjectRoot to return false in a temp directory with no go.mod")
	}
}

func TestInitConfig_MkdirError(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	blockedPath := filepath.Join(tmpDir, "blocked")
	if err := os.WriteFile(blockedPath, []byte("file"), 0644); err != nil {
		t.Fatalf("failed to write blocked file: %v", err)
	}

	err := initConfig(filepath.Join(blockedPath, "config.json"))
	if err == nil {
		t.Error("expected error for blocked config directory creation, got nil")
	}
}

func TestInitConfig_DirectoryPath(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()

	err := initConfig(tmpDir)
	if err == nil {
		t.Error("expected error when passing directory path as config file, got nil")
	}
}

func TestInitConfig_InvalidJSON(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(cfgPath, []byte("invalid-json{"), 0644); err != nil {
		t.Fatalf("failed to write invalid json: %v", err)
	}

	err := initConfig(cfgPath)
	if err == nil {
		t.Error("expected error for invalid config file json, got nil")
	}
}

func TestEnsureSchema(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	// 1. Config path is empty
	cfg := &Config{v: viper.New()}
	if err := cfg.ensureSchema(); err != nil {
		t.Errorf("expected no error for empty config path, got %v", err)
	}

	// 2. Config file does not exist
	cfg.v.SetConfigFile(cfgPath)
	if err := cfg.ensureSchema(); err != nil {
		t.Errorf("expected no error for non-existent config file, got %v", err)
	}

	// 3. Config file contains invalid JSON
	if err := os.WriteFile(cfgPath, []byte("invalid"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := cfg.ensureSchema(); err != nil {
		t.Errorf("expected no error for invalid JSON content in ensureSchema, got %v", err)
	}

	// 4. Config file already has $schema
	validWithSchema := `{"$schema": "some-url", "vault": {}}`
	if err := os.WriteFile(cfgPath, []byte(validWithSchema), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := cfg.ensureSchema(); err != nil {
		t.Errorf("expected no error when $schema already exists, got %v", err)
	}

	// 5. Config file does not have $schema - injects it
	validWithoutSchema := `{"vault": {}}`
	if err := os.WriteFile(cfgPath, []byte(validWithoutSchema), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := cfg.ensureSchema(); err != nil {
		t.Errorf("expected no error during schema injection, got %v", err)
	}

	// Verify $schema was injected
	content, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(content, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if m["$schema"] != SchemaURL {
		t.Errorf("expected injected schema %s, got %v", SchemaURL, m["$schema"])
	}
}

func TestRunWizard_FormError(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	err := initConfig(cfgPath)
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	originalRunForm := runWizardForm
	defer func() { runWizardForm = originalRunForm }()
	runWizardForm = func(form *huh.Form) error {
		return fmt.Errorf("form cancelled")
	}

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	_, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() { _ = wOut.Close() }()

	err = RunWizard()
	if err == nil {
		t.Error("expected error from form cancelled, got nil")
	}
}

func TestDaemonReload(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"gitsync": {"enabled": true}}`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := initConfig(configPath); err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	d := BootstrapDaemon()
	if d == nil {
		t.Fatal("BootstrapDaemon returned nil")
	}

	// Trigger reload callback manually to cover the OnReload callback in NewDaemon
	cfg := GetConfig()
	cfg.mu.Lock()
	callbacks := make([]func(*Config), len(cfg.onReload))
	copy(callbacks, cfg.onReload)
	cfg.mu.Unlock()

	for _, cb := range callbacks {
		cb(cfg)
	}
}
