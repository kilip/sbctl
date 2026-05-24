package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kilip/sbctl/internal/config"
	"github.com/kilip/sbctl/internal/gitsync"
)

type Daemon struct {
	workers []Worker
	logger  *slog.Logger
	mu      sync.Mutex
	cancel  context.CancelFunc
	ctx     context.Context
}

func NewDaemon() *Daemon {
	return &Daemon{
		logger: slog.Default().With("module", "daemon"),
	}
}

func (d *Daemon) Start() error {
	d.logger.Info("starting sbctl daemon")

	if err := d.writePID(); err != nil {
		return fmt.Errorf("failed to write PID: %w", err)
	}
	defer d.removePID()

	if err := d.setupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	d.ctx, d.cancel = context.WithCancel(context.Background())
	defer d.cancel()

	// Initial worker start
	d.reloadWorkers()

	// Watch for config changes
	go d.watchConfig()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		d.logger.Info("received signal, shutting down", "signal", sig)
	case <-d.ctx.Done():
		d.logger.Info("daemon context cancelled")
	}

	return nil
}

func (d *Daemon) reloadWorkers() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop existing workers by cancelling context
	if d.cancel != nil {
		d.cancel()
	}
	d.ctx, d.cancel = context.WithCancel(context.Background())

	cfg := config.GetConfig()
	d.workers = []Worker{
		gitsync.NewGitSync(&cfg.GitSync),
		// Add other workers here as they are implemented
	}

	for _, w := range d.workers {
		d.logger.Info("starting worker", "name", w.Name())
		go func(worker Worker) {
			if err := worker.Start(d.ctx); err != nil {
				d.logger.Error("worker failed", "name", worker.Name(), "error", err)
			}
		}(w)
	}
}

func (d *Daemon) watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		d.logger.Error("failed to create config watcher", "error", err)
		return
	}
	defer watcher.Close()

	configPath := d.getConfigFilePath()
	if err := watcher.Add(filepath.Dir(configPath)); err != nil {
		d.logger.Error("failed to watch config directory", "error", err)
		return
	}

	d.logger.Info("watching config for changes", "path", configPath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Name == configPath && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) {
				d.logger.Info("config change detected, reloading workers")
				// Debounce reload
				time.Sleep(500 * time.Millisecond)
				d.reloadWorkers()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			d.logger.Error("config watcher error", "error", err)
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Daemon) writePID() error {
	pidPath := filepath.Join(getConfigDir(), "sbctl.pid")
	pid := os.Getpid()
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
}

func (d *Daemon) removePID() {
	pidPath := filepath.Join(getConfigDir(), "sbctl.pid")
	_ = os.Remove(pidPath)
}

func (d *Daemon) setupLogging() error {
	logPath := filepath.Join(getConfigDir(), "sbctl.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	// Note: In a real implementation, we might want to use a multi-writer
	// or properly configure the logger to write to this file.
	// For simplicity, we'll assume the logger is already writing to stdout
	// and we redirect stdout/stderr to the file if running as daemon.
	os.Stdout = f
	os.Stderr = f
	return nil
}

func (d *Daemon) getConfigFilePath() string {
	// Logic similar to config.initConfig()
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sbctl", "config.json")
}
