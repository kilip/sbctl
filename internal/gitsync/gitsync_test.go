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

func TestGitSync_RunGitError(t *testing.T) {
	gs := NewGitSync(&Config{Dir: "/invalid/dir"})
	err := gs.runGit("status")
	if err == nil {
		t.Error("expected error from runGit on invalid dir, got nil")
	}

	_, err = gs.runGitOutput("status")
	if err == nil {
		t.Error("expected error from runGitOutput on invalid dir, got nil")
	}
}

func TestGitSync_StartErrors(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()

	tmpDir := t.TempDir()

	// Case 1: runGit("init") fails
	cfg := &Config{
		Dir:     tmpDir,
		Enabled: true,
	}
	gs := NewGitSync(cfg)

	ExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "git" && len(args) > 0 && args[0] == "init" {
			return exec.Command("false")
		}
		return exec.Command(name, args...)
	}

	err := gs.Start(context.Background())
	if err == nil {
		t.Error("expected error when git init fails, got nil")
	}

	// Case 2: runGitOutput("remote") fails
	// First initialize .git directory so it skips init
	_ = os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)
	cfg.GitRepository = "git@github.com:example/repo.git"

	ExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "git" && len(args) > 0 && args[0] == "remote" {
			return exec.Command("false")
		}
		return exec.Command(name, args...)
	}

	err = gs.Start(context.Background())
	if err == nil {
		t.Error("expected error when git remote fails, got nil")
	}

	// Case 3: runGit("remote", "add", ...) fails
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "git" && len(args) > 0 {
			if args[0] == "remote" && len(args) == 1 {
				// Success, returns empty output (no origin)
				return exec.Command("true")
			}
			if args[0] == "remote" && len(args) > 1 && args[1] == "add" {
				return exec.Command("false")
			}
		}
		return exec.Command(name, args...)
	}

	err = gs.Start(context.Background())
	if err == nil {
		t.Error("expected error when remote add fails, got nil")
	}
}

func TestGitSync_SyncErrors(t *testing.T) {
	originalExecCommand := ExecCommand
	defer func() { ExecCommand = originalExecCommand }()

	tmpDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)

	cfg := &Config{
		Dir:           tmpDir,
		Enabled:       true,
		GitRepository: "git@github.com:example/repo.git",
	}

	var failCmd string
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "git" && len(args) > 0 {
			if args[0] == failCmd {
				return exec.Command("false")
			}
			if args[0] == "status" {
				// return modified status
				return exec.Command("echo", "M main.go")
			}
			if args[0] == "rev-parse" {
				return exec.Command("echo", "main")
			}
		}
		return exec.Command(name, args...)
	}

	gs := NewGitSync(cfg)

	// Case 1: git add fails
	failCmd = "add"
	err := gs.Sync()
	if err == nil {
		t.Error("expected error when git add fails, got nil")
	}

	// Case 2: git status fails
	failCmd = "status"
	err = gs.Sync()
	if err == nil {
		t.Error("expected error when git status fails, got nil")
	}

	// Case 3: git commit fails
	failCmd = "commit"
	err = gs.Sync()
	if err == nil {
		t.Error("expected error when git commit fails, got nil")
	}

	// Case 4: git rev-parse fails
	failCmd = "rev-parse"
	err = gs.Sync()
	if err == nil {
		t.Error("expected error when git rev-parse fails, got nil")
	}

	// Case 5: git pull fails
	failCmd = "pull"
	err = gs.Sync()
	if err == nil {
		t.Error("expected error when git pull fails, got nil")
	}

	// Case 6: git push fails
	failCmd = "push"
	err = gs.Sync()
	if err == nil {
		t.Error("expected error when git push fails, got nil")
	}
}
