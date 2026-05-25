package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type mockPlatformManager struct {
	installCalled   bool
	uninstallCalled bool
	isInstalledVal  bool
}

func (m *mockPlatformManager) Install(binPath string) error {
	m.installCalled = true
	return nil
}

func (m *mockPlatformManager) Uninstall() error {
	m.uninstallCalled = true
	return nil
}

func (m *mockPlatformManager) IsInstalled() (bool, error) {
	return m.isInstalledVal, nil
}

// TestHelperProcess is used to mock execCommand.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(2)
	}

	cmd := args[0]
	switch cmd {
	case "tail", "powershell":
		fmt.Println("mock tail logs line 1")
		fmt.Println("mock tail logs line 2")
		os.Exit(0)
	}
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestNewManager(t *testing.T) {
	// Simple validation
	m, err := NewManager("/tmp", "/tmp/config.json")
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestManager_InstallUninstall(t *testing.T) {
	mockPM := &mockPlatformManager{isInstalledVal: true}
	m := &Manager{
		platform:   mockPM,
		configDir:  "/tmp",
		configFile: "/tmp/config.json",
	}

	if err := m.Install(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !mockPM.installCalled {
		t.Error("expected platform Install to be called")
	}

	if err := m.Uninstall(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !mockPM.uninstallCalled {
		t.Error("expected platform Uninstall to be called")
	}
}

func TestManager_Status(t *testing.T) {
	tmpDir := t.TempDir()
	mockPM := &mockPlatformManager{}
	m := &Manager{
		platform:  mockPM,
		configDir: tmpDir,
	}

	// Status Stopped (no PID file)
	if err := m.Status(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Create invalid PID file
	pidFile := filepath.Join(tmpDir, "sbctl.pid")
	if err := os.WriteFile(pidFile, []byte("invalid-pid"), 0644); err != nil {
		t.Fatalf("failed to write pid file: %v", err)
	}

	if err := m.Status(); err == nil {
		t.Error("expected error for invalid PID, got nil")
	}

	// Create valid PID for non-existent process
	if err := os.WriteFile(pidFile, []byte("999999"), 0644); err != nil {
		t.Fatalf("failed to write pid file: %v", err)
	}
	if err := m.Status(); err != nil {
		t.Errorf("expected no error for non-existent PID, got %v", err)
	}
}

func TestManager_Info(t *testing.T) {
	tmpDir := t.TempDir()
	mockPM := &mockPlatformManager{isInstalledVal: true}
	m := &Manager{
		platform:   mockPM,
		configDir:  tmpDir,
		configFile: filepath.Join(tmpDir, "config.json"),
	}

	if err := m.Info(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestManager_Logs(t *testing.T) {
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()
	execCommand = fakeExecCommand

	tmpDir := t.TempDir()
	mockPM := &mockPlatformManager{}
	m := &Manager{
		platform:  mockPM,
		configDir: tmpDir,
	}

	if err := m.Logs(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestManager_Stop_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	mockPM := &mockPlatformManager{}
	m := &Manager{
		platform:  mockPM,
		configDir: tmpDir,
	}

	err := m.Stop()
	if err == nil {
		t.Error("expected error when stopping non-running service, got nil")
	}
}

func TestManager_StartStopSuccess(t *testing.T) {
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "sbctl.pid")

	execCommand = func(name string, args ...string) *exec.Cmd {
		// Mock the daemon process by running sh that writes its PID without newline
		cmdStr := fmt.Sprintf("printf \"%%d\" $$ > %s && sleep 10", pidPath)
		return exec.Command("sh", "-c", cmdStr)
	}

	mockPM := &mockPlatformManager{}
	m := &Manager{
		platform:  mockPM,
		configDir: tmpDir,
	}

	// Start service
	err := m.Start()
	if err != nil {
		t.Fatalf("expected no error starting manager, got %v", err)
	}

	// Verify PID file exists and is non-empty
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Error("expected PID file to exist")
	}

	// Stop service
	err = m.Stop()
	if err != nil {
		t.Fatalf("expected no error stopping manager, got %v", err)
	}

	// Verify PID file is gone
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("expected PID file to be removed after Stop")
	}
}

func TestManager_Restart(t *testing.T) {
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "sbctl.pid")

	execCommand = func(name string, args ...string) *exec.Cmd {
		cmdStr := fmt.Sprintf("printf \"%%d\" $$ > %s && sleep 10", pidPath)
		return exec.Command("sh", "-c", cmdStr)
	}

	mockPM := &mockPlatformManager{}
	m := &Manager{
		platform:  mockPM,
		configDir: tmpDir,
	}

	// Restart (which will try to stop first and warn, then start)
	err := m.Restart()
	if err != nil {
		t.Fatalf("expected no error restarting manager, got %v", err)
	}

	// Verify PID file exists
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Error("expected PID file to exist after Restart")
	}

	// Stop it
	_ = m.Stop()
}

func TestStubs(t *testing.T) {
	dm := &DarwinManager{}
	_ = dm.Install("")
	_ = dm.Uninstall()
	_, _ = dm.IsInstalled()

	wm := &WindowsManager{}
	_ = wm.Install("")
	_ = wm.Uninstall()
	_, _ = wm.IsInstalled()
}
