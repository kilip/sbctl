//go:build darwin

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

func (d *DarwinManager) Install(binPath string) error {
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.sbctl.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`, binPath)

	dir := filepath.Join(os.Getenv("HOME"), "Library/LaunchAgents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "com.sbctl.daemon.plist")
	return os.WriteFile(path, []byte(content), 0644)
}

func (d *DarwinManager) Uninstall() error {
	path := filepath.Join(os.Getenv("HOME"), "Library/LaunchAgents/com.sbctl.daemon.plist")
	return os.Remove(path)
}

func (d *DarwinManager) IsInstalled() (bool, error) {
	path := filepath.Join(os.Getenv("HOME"), "Library/LaunchAgents/com.sbctl.daemon.plist")
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
