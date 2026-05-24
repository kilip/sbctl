//go:build !windows

package daemon

import "errors"

func (w *WindowsManager) Install(binPath string) error {
	return errors.New("not implemented on this platform")
}

func (w *WindowsManager) Uninstall() error {
	return errors.New("not implemented on this platform")
}

func (w *WindowsManager) IsInstalled() (bool, error) {
	return false, errors.New("not implemented on this platform")
}
