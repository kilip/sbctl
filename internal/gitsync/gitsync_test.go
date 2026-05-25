package gitsync

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestGitSync_StartAndSync(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Dir:      tmpDir,
		Enabled:  true,
		Debounce: 10 * time.Millisecond,
	}

	gs := NewGitSync(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gs.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Set git identity for test environment
	setupGitIdentity(t, tmpDir)

	// Check if git repo was initialized
	if _, err := os.Stat(filepath.Join(tmpDir, ".git")); os.IsNotExist(err) {
		t.Errorf("expected git repo to be initialized")
	}

	// Create a file to trigger sync
	err = os.WriteFile(tmpDir+"/test.txt", []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Wait for debounce and sync
	time.Sleep(200 * time.Millisecond)

	// Verify commit
	cmd := exec.Command("git", "log", "-1")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("expected at least one commit: %v. Output: %s", err, string(out))
	}
}

func setupGitIdentity(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set git user.email: %v", err)
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set git user.name: %v", err)
	}
}

func TestGitSync_Name(t *testing.T) {
	gs := NewGitSync(&Config{})
	if gs.Name() != "gitsync" {
		t.Errorf("expected gitsync, got %s", gs.Name())
	}
}

func TestGitSync_Disabled(t *testing.T) {
	gs := NewGitSync(&Config{Enabled: false})
	err := gs.Start(context.Background())
	if err != nil {
		t.Fatalf("expected no error for disabled gitsync, got %v", err)
	}
}

func TestGitSync_EmptyDir(t *testing.T) {
	gs := NewGitSync(&Config{Enabled: true, Dir: ""})
	err := gs.Start(context.Background())
	if err == nil {
		t.Error("expected error for empty directory, got nil")
	}
}

func TestGitSync_RemoteSync(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()

	tmpDir := t.TempDir()

	// Initialize git manually first
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to init git: %v", err)
	}

	setupGitIdentity(t, tmpDir)

	cfg := &Config{
		Dir:           tmpDir,
		Enabled:       true,
		GitRepository: "git@github.com:example/repo.git",
		Debounce:      10 * time.Millisecond,
	}

	gs := NewGitSync(cfg)

	// Mock git commands for remote operations
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		// Mock git pull/push/rev-parse remote commands to avoid network access
		if name == "git" && len(args) > 0 {
			switch args[0] {
			case "pull", "push", "remote":
				// Return true command (exit 0)
				return exec.Command("true")
			case "rev-parse":
				// Return main branch
				return exec.Command("echo", "main")
			}
		}
		// Delegate normal commands
		return exec.Command(name, args...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gs.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Trigger sync manually via Sync
	err = gs.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
}
