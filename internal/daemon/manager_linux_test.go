//go:build linux

package daemon

import (
	"os"
	"runtime"
	"testing"
)

func TestLinuxManager(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-only test")
	}
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	lm := &LinuxManager{}
	err := lm.Install("/usr/local/bin/sbctl")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	installed, err := lm.IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled failed: %v", err)
	}
	if !installed {
		t.Error("expected installed to be true")
	}

	err = lm.Uninstall()
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	installed, err = lm.IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled failed: %v", err)
	}
	if installed {
		t.Error("expected installed to be false")
	}
}
