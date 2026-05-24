package daemon

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

type mockWorker struct {
	name    string
	started bool
}

func (m *mockWorker) Name() string {
	return m.name
}

func (m *mockWorker) Start(ctx context.Context) error {
	m.started = true
	<-ctx.Done()
	return nil
}

func TestDaemon_StartAndReload(t *testing.T) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	oldDefault := slog.Default()
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		slog.SetDefault(oldDefault)
	}()

	worker1 := &mockWorker{name: "worker1"}
	worker2 := &mockWorker{name: "worker2"}

	provider := func() []Worker {
		return []Worker{worker1, worker2}
	}

	// Use a temp file for config path
	configPath := t.TempDir() + "/config.json"

	d := NewDaemon(provider, configPath)

	// Use a context with timeout to ensure the test doesn't run forever
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start daemon in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Start()
	}()

	// Wait for workers to start
	time.Sleep(100 * time.Millisecond)

	if !worker1.started {
		t.Error("worker1 was not started")
	}
	if !worker2.started {
		t.Error("worker2 was not started")
	}

	// Trigger reload
	d.reloadWorkers()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// In a real scenario, reloadWorkers creates NEW worker instances if the provider does.
	// Our mock provider returns the same pointers, so they should still be "started" (though the old context was cancelled)
	// Actually, the previous workers' Start methods would return when their context is cancelled.

	// Stop the daemon
	d.cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("daemon stopped with error: %v", err)
		}
	case <-ctx.Done():
		t.Error("test timed out waiting for daemon to stop")
	}
}
