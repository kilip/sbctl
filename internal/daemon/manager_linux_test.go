//go:build linux

package daemon

import (
	"runtime"
	"testing"
)

func TestLinuxManager(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-only test")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

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
