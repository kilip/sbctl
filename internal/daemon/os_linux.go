//go:build linux

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

func (l *LinuxManager) Install(binPath string) error {
	content := fmt.Sprintf(`[Unit]
Description=sbctl daemon
After=network.target

[Service]
ExecStart=%s daemon
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
`, binPath)

	dir := filepath.Join(os.Getenv("HOME"), ".config/systemd/user")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "sbctl.service")
	return os.WriteFile(path, []byte(content), 0644)
}

func (l *LinuxManager) Uninstall() error {
	path := filepath.Join(os.Getenv("HOME"), ".config/systemd/user/sbctl.service")
	return os.Remove(path)
}

func (l *LinuxManager) IsInstalled() (bool, error) {
	path := filepath.Join(os.Getenv("HOME"), ".config/systemd/user/sbctl.service")
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
