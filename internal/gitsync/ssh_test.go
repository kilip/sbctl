package gitsync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestHelperProcess is used to mock execCommand.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd := args[0]
	switch cmd {
	case "gh":
		// Check arguments
		if len(args) >= 3 && args[1] == "auth" && args[2] == "status" {
			os.Exit(0)
		}
		if len(args) >= 4 && args[1] == "ssh-key" && args[2] == "add" {
			// If title contains "signing", simulate success or signing add
			os.Exit(0)
		}
	case "git":
		if len(args) >= 2 && args[1] == "init" {
			os.Exit(0)
		}
		if len(args) >= 4 && args[1] == "config" {
			os.Exit(0)
		}
	}
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestConfigureSSH_Success(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()

	ExecCommand = fakeExecCommand

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	vaultDir := filepath.Join(tmpDir, "vault")

	// Ensure vault git directory is simulated
	err := os.MkdirAll(filepath.Join(vaultDir, ".git"), 0755)
	if err != nil {
		t.Fatalf("failed to create fake git dir: %v", err)
	}

	err = ConfigureSSH(configDir, vaultDir, "test@example.com")
	if err != nil {
		t.Fatalf("ConfigureSSH failed: %v", err)
	}

	// Verify key files exist
	keyPath := filepath.Join(configDir, ".ssh", "id_ed25519")
	pubKeyPath := keyPath + ".pub"

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("expected private key file %s to exist", keyPath)
	}
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		t.Errorf("expected public key file %s to exist", pubKeyPath)
	}
}

func TestConfigureSSH_MissingParams(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")

	err := ConfigureSSH(configDir, "", "test@example.com")
	if err == nil {
		t.Error("expected error for empty vaultDir, got nil")
	}

	err = ConfigureSSH(configDir, "/tmp/vault", "")
	if err == nil {
		t.Error("expected error for empty userEmail, got nil")
	}
}

func TestConfigureSSH_GHError(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()

	// Force failure of gh auth status
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "gh" && args[0] == "auth" && args[1] == "status" {
			cs := []string{"-test.run=TestHelperProcess", "--", "false"}
			cmd := exec.Command(os.Args[0], cs...)
			cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
			return cmd
		}
		return fakeExecCommand(command, args...)
	}

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	vaultDir := filepath.Join(tmpDir, "vault")

	err := ConfigureSSH(configDir, vaultDir, "test@example.com")
	if err == nil {
		t.Error("expected error due to gh auth status failure, got nil")
	}
}

func TestConfigureSSH_MkdirError(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()
	ExecCommand = fakeExecCommand

	tmpDir := t.TempDir()
	blockedPath := filepath.Join(tmpDir, "blocked")
	if err := os.WriteFile(blockedPath, []byte("file"), 0644); err != nil {
		t.Fatalf("failed to write blocked file: %v", err)
	}

	err := ConfigureSSH(blockedPath, "/tmp/vault", "test@example.com")
	if err == nil {
		t.Error("expected error due to blocked directory creation, got nil")
	}
}
