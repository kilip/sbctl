package daemon

type PlatformManager interface {
	Install(binPath string) error
	Uninstall() error
	IsInstalled() (bool, error)
}

// Dummy structures to ensure Manager compiles on all platforms
// although only one will be active via build tags.

type LinuxManager struct{}
type DarwinManager struct{}
type WindowsManager struct{}
