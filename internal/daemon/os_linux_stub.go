//go:build !linux

package daemon

import "errors"

func (l *LinuxManager) Install(binPath string) error {
	return errors.New("not implemented on this platform")
}

func (l *LinuxManager) Uninstall() error {
	return errors.New("not implemented on this platform")
}

func (l *LinuxManager) IsInstalled() (bool, error) {
	return false, errors.New("not implemented on this platform")
}
