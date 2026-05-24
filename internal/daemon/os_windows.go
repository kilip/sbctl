//go:build windows

package daemon

import (
	"fmt"
	"os/exec"
)

func (w *WindowsManager) Install(binPath string) error {
	// HKCU\Software\Microsoft\Windows\CurrentVersion\Run
	cmd := exec.Command("reg", "add", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "sbctl", "/t", "REG_SZ", "/d", fmt.Sprintf("\"%s\" daemon", binPath), "/f")
	return cmd.Run()
}

func (w *WindowsManager) Uninstall() error {
	cmd := exec.Command("reg", "delete", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "sbctl", "/f")
	return cmd.Run()
}

func (w *WindowsManager) IsInstalled() (bool, error) {
	cmd := exec.Command("reg", "query", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "sbctl")
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	return false, nil
}
