//go:build !darwin

package daemon

import "errors"

func (d *DarwinManager) Install(binPath string) error {
	return errors.New("not implemented on this platform")
}

func (d *DarwinManager) Uninstall() error {
	return errors.New("not implemented on this platform")
}

func (d *DarwinManager) IsInstalled() (bool, error) {
	return false, errors.New("not implemented on this platform")
}
