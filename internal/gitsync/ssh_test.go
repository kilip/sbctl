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

func TestConfigureSSH_VariousErrors(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	vaultDir := filepath.Join(tmpDir, "vault")
	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		t.Fatalf("failed to create vaultDir: %v", err)
	}

	var mockAction string
	ExecCommand = func(command string, args ...string) *exec.Cmd {
		if command == "gh" && args[0] == "auth" {
			return exec.Command("true")
		}
		if mockAction == "gh_ssh_add_fail" && command == "gh" && args[0] == "ssh-key" {
			return exec.Command("false")
		}
		if mockAction == "gh_ssh_add_already_exists" && command == "gh" && args[0] == "ssh-key" {
			return exec.Command("sh", "-c", "echo 'error: key already in use'; exit 1")
		}
		if mockAction == "git_init_fail" && command == "git" && args[0] == "init" {
			return exec.Command("false")
		}
		if mockAction == "git_config_fail" && command == "git" && args[0] == "config" {
			return exec.Command("false")
		}
		return exec.Command("true")
	}

	// Case 1a: gh ssh-key add fails generally
	mockAction = "gh_ssh_add_fail"
	err := ConfigureSSH(filepath.Join(configDir, "1a"), vaultDir, "test@example.com")
	if err != nil {
		t.Errorf("expected warning only, but got error: %v", err)
	}

	// Case 1b: gh ssh-key add fails but "already in use"
	mockAction = "gh_ssh_add_already_exists"
	err = ConfigureSSH(filepath.Join(configDir, "1b"), vaultDir, "test@example.com")
	if err != nil {
		t.Errorf("expected warning only, but got error: %v", err)
	}

	// Case 2: git init fails
	mockAction = "git_init_fail"
	vaultDirInitFail := filepath.Join(tmpDir, "vault_init_fail")
	if err := os.MkdirAll(vaultDirInitFail, 0755); err != nil {
		t.Fatalf("failed to create vaultDirInitFail: %v", err)
	}
	err = ConfigureSSH(filepath.Join(configDir, "2"), vaultDirInitFail, "test@example.com")
	if err == nil {
		t.Error("expected error due to git init failure, got nil")
	}

	// Case 3: git config fails
	mockAction = "git_config_fail"
	vaultDirConfigFail := filepath.Join(tmpDir, "vault_config_fail")
	_ = os.MkdirAll(filepath.Join(vaultDirConfigFail, ".git"), 0755)
	err = ConfigureSSH(filepath.Join(configDir, "3"), vaultDirConfigFail, "test@example.com")
	if err == nil {
		t.Error("expected error due to git config failure, got nil")
	}
}

func TestGenerateEd25519Key_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	blockedPath := filepath.Join(tmpDir, "blocked_dir")
	err := os.MkdirAll(blockedPath, 0755)
	if err != nil {
		t.Fatalf("failed to create blocked dir: %v", err)
	}

	// 1. Private key path is a directory (should fail OpenFile)
	err = generateEd25519Key(blockedPath, filepath.Join(tmpDir, "key.pub"), "test@example.com")
	if err == nil {
		t.Error("expected error when private key path is a directory, got nil")
	}

	// 2. Public key path is a directory (should fail os.WriteFile)
	validKeyPath := filepath.Join(tmpDir, "key")
	err = generateEd25519Key(validKeyPath, blockedPath, "test@example.com")
	if err == nil {
		t.Error("expected error when public key path is a directory, got nil")
	}
}
