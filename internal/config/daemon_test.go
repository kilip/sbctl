package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapDaemon(t *testing.T) {
	Reset()
	testCwd, _ := os.Getwd()
	configDir := filepath.Join(testCwd, ".sbctl")
	configPath := filepath.Join(configDir, "config.json")

	// Ensure cleanup
	defer func() { _ = os.RemoveAll(configDir) }()

	// Create a dummy config file
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"gitsync": {"enabled": true}}`), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Initialize config
	if err := initConfig(configPath); err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	d := BootstrapDaemon()
	if d == nil {
		t.Fatal("BootstrapDaemon returned nil")
	}

	// Verify that workers are loaded via the provider
	// We can't easily check private fields of Daemon, but we can call reloadWorkers
	// (though it's private, we are in the same package... wait, no, daemon is in internal/daemon)

	// Actually, NewDaemon and BootstrapDaemon are exported, so we can check if it returns a valid daemon.
}
