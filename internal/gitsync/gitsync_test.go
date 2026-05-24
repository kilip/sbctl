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
