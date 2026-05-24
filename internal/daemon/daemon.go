package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kilip/sbctl/internal/shared/logger"
)

type WorkerProvider func() []Worker

type Daemon struct {
	workers        []Worker
	workerProvider WorkerProvider
	configPath     string
	logger         logger.Logger
	mu             sync.Mutex
	cancel         context.CancelFunc
	ctx            context.Context
	workerCancel   context.CancelFunc
}

func NewDaemon(provider WorkerProvider, configPath string, l logger.Logger) *Daemon {
	return &Daemon{
		logger:         l.With("module", "daemon"),
		workerProvider: provider,
		configPath:     configPath,
	}
}

func (d *Daemon) Start() error {
	if err := d.setupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	d.logger.Info("starting sbctl daemon")

	if err := d.writePID(); err != nil {
		return fmt.Errorf("failed to write PID: %w", err)
	}
	defer d.removePID()

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
	if d.workerCancel != nil {
		d.workerCancel()
	}
	var workerCtx context.Context
	workerCtx, d.workerCancel = context.WithCancel(d.ctx)

	d.workers = d.workerProvider()

	for _, w := range d.workers {
		d.logger.Info("starting worker", "name", w.Name())
		go func(worker Worker) {
			if err := worker.Start(workerCtx); err != nil {
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

	if err := watcher.Add(filepath.Dir(d.configPath)); err != nil {
		d.logger.Error("failed to watch config directory", "error", err)
		return
	}

	d.logger.Info("watching config for changes", "path", d.configPath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Name == d.configPath && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) {
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
	pidPath := filepath.Join(filepath.Dir(d.configPath), "sbctl.pid")
	pid := os.Getpid()
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
}

func (d *Daemon) removePID() {
	pidPath := filepath.Join(filepath.Dir(d.configPath), "sbctl.pid")
	_ = os.Remove(pidPath)
}

func (d *Daemon) setupLogging() error {
	logPath := filepath.Join(filepath.Dir(d.configPath), "sbctl.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Redirect stdout/stderr to the file
	os.Stdout = f
	os.Stderr = f

	// Update logger to write to the file
	d.logger = logger.NewSlogLoggerWithWriter("info", f).With("module", "daemon")

	return nil
}
