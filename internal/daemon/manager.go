package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

// Manager handles service operations (install, uninstall, start, stop, logs).
type Manager struct {
	platform   PlatformManager
	configDir  string
	configFile string
}

// NewManager creates a new service manager based on the current OS.
func NewManager(configDir string, configFile string) (*Manager, error) {
	var pm PlatformManager
	switch runtime.GOOS {
	case "linux":
		pm = &LinuxManager{}
	case "darwin":
		pm = &DarwinManager{}
	case "windows":
		pm = &WindowsManager{}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return &Manager{
		platform:   pm,
		configDir:  configDir,
		configFile: configFile,
	}, nil
}

func (m *Manager) Install() error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	return m.platform.Install(binPath)
}

func (m *Manager) Uninstall() error {
	return m.platform.Uninstall()
}

func (m *Manager) Start() error {
	// Logic to start the background process depends on the OS,
	// but generally we want to ensure it's installed first.
	return m.startProcess()
}

func (m *Manager) Stop() error {
	return m.stopProcess()
}

func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		// If it's not running, just start it
		fmt.Printf("Warning: %v. Starting anyway...\n", err)
	}
	return m.Start()
}

func (m *Manager) Status() error {
	pidPath := filepath.Join(m.configDir, "sbctl.pid")
	b, err := os.ReadFile(pidPath)
	if err != nil {
		fmt.Println("Status: Stopped")
		return nil
	}

	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return fmt.Errorf("invalid PID in file")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("Status: Stopped (Stale PID file)")
		return nil
	}

	// On Unix, FindProcess always succeeds, we need to signal 0 to check existence
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		fmt.Println("Status: Stopped (Process not found)")
		return nil
	}

	fmt.Printf("Status: Running (PID: %d)\n", pid)
	return nil
}

func (m *Manager) Info() error {
	fmt.Printf("Config Directory: %s\n", m.configDir)
	fmt.Printf("Config File:      %s\n", m.configFile)
	logPath := filepath.Join(m.configDir, "sbctl.log")
	fmt.Printf("Log File:         %s\n", logPath)
	pidPath := filepath.Join(m.configDir, "sbctl.pid")
	fmt.Printf("PID File:         %s\n", pidPath)

	installed, err := m.platform.IsInstalled()
	if err != nil {
		fmt.Printf("Service Status:   Unknown (%v)\n", err)
	} else if installed {
		fmt.Println("Service Status:   Installed")
	} else {
		fmt.Println("Service Status:   Not Installed")
	}

	return nil
}

func (m *Manager) Logs() error {
	logPath := filepath.Join(m.configDir, "sbctl.log")
	// Simple tail implementation
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows doesn't have tail by default, we'll need a different approach later
		// for now, just print the path or use powershell
		cmd = exec.Command("powershell", "Get-Content", logPath, "-Wait")
	} else {
		cmd = exec.Command("tail", "-f", logPath)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) startProcess() error {
	pidPath := filepath.Join(m.configDir, "sbctl.pid")
	if b, _ := os.ReadFile(pidPath); len(b) > 0 {
		pid, _ := strconv.Atoi(string(b))
		if proc, err := os.FindProcess(pid); err == nil {
			// Check if process is actually running
			if err := proc.Signal(syscall.Signal(0)); err == nil {
				return fmt.Errorf("service is already running (PID: %d)", pid)
			}
		}
	}

	binPath, _ := os.Executable()
	args := []string{"daemon"}
	if m.configFile != "" {
		args = append(args, "--config", m.configFile)
	}
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = nil // Will be handled by daemon's setupLogging
	cmd.Stderr = nil

	// Platform specific detachment
	detachProcess(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to release process: %w", err)
	}

	// Poll for PID file to confirm startup
	fmt.Print("Starting service...")
	success := false
	for i := 0; i < 30; i++ { // 3 seconds timeout
		if b, err := os.ReadFile(pidPath); err == nil {
			currentPid, _ := strconv.Atoi(string(b))
			if currentPid == pid {
				success = true
				break
			}
		}
		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	if !success {
		return fmt.Errorf("\nfailed to confirm service startup (PID file not found)")
	}

	fmt.Printf("\nService started with PID %d\n", pid)
	return nil
}

func (m *Manager) stopProcess() error {
	pidPath := filepath.Join(m.configDir, "sbctl.pid")
	b, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("service is not running (no PID file)")
	}

	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return fmt.Errorf("invalid PID in file")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try to stop gracefully
	var errSignal error
	if runtime.GOOS == "windows" {
		errSignal = proc.Kill()
	} else {
		errSignal = proc.Signal(syscall.SIGTERM)
	}

	if errSignal != nil {
		return fmt.Errorf("failed to stop process: %w", errSignal)
	}

	// Wait for PID file to be removed (daemon's cleanup)
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(pidPath); os.IsNotExist(err) {
			fmt.Println("Service stopped.")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force remove if it still exists after timeout
	_ = os.Remove(pidPath)
	fmt.Printf("Service (PID: %d) stopped (forced cleanup)\n", pid)
	return nil
}
