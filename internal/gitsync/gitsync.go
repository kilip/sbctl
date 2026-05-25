package gitsync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type GitSync struct {
	config *Config
	mu     sync.Mutex
	timer  *time.Timer
	logger *slog.Logger
}

// NewGitSync creates a new GitSync instance with the given config.
func NewGitSync(cfg *Config) *GitSync {
	return &GitSync{
		config: cfg,
		logger: slog.Default().With("module", "gitsync"),
	}
}

// Name returns the name of the worker.
func (gs *GitSync) Name() string {
	return "gitsync"
}

// Start begins watching the directory for changes.
func (gs *GitSync) Start(ctx context.Context) error {
	if !gs.config.Enabled {
		gs.logger.Info("gitsync disabled")
		return nil
	}

	dir := gs.config.Dir
	if dir == "" {
		return fmt.Errorf("directory not configured")
	}

	// Ensure it's a git repo
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		gs.logger.Info("initializing git repository", "path", dir)
		if err := gs.runGit("init"); err != nil {
			return fmt.Errorf("failed to init git repo: %w", err)
		}
	}

	// Handle remote
	if gs.config.GitRepository != "" {
		remotes, err := gs.runGitOutput("remote")
		if err != nil {
			return fmt.Errorf("failed to check remotes: %w", err)
		}

		if !strings.Contains(remotes, "origin") {
			gs.logger.Info("adding remote origin", "url", gs.config.GitRepository)
			if err := gs.runGit("remote", "add", "origin", gs.config.GitRepository); err != nil {
				return fmt.Errorf("failed to add remote: %w", err)
			}
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Recursive watch
	if err := gs.watchRecursive(watcher, dir); err != nil {
		_ = watcher.Close()
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	gs.logger.Info("gitsync started", "path", dir, "debounce", gs.config.Debounce)

	go gs.watchLoop(ctx, watcher)

	return nil
}

func (gs *GitSync) watchLoop(ctx context.Context, watcher *fsnotify.Watcher) {
	defer func() { _ = watcher.Close() }()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Skip .git directory
			if isGitInternal(event.Name) {
				continue
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				gs.logger.Debug("file change detected", "file", event.Name, "op", event.Op)
				gs.triggerSync()

				// If new directory created, watch it
				if event.Has(fsnotify.Create) {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						_ = gs.watchRecursive(watcher, event.Name)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			gs.logger.Error("watcher error", "error", err)
		case <-ctx.Done():
			gs.logger.Info("gitsync stopping")
			return
		}
	}
}

func (gs *GitSync) triggerSync() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.timer != nil {
		gs.timer.Stop()
	}

	gs.timer = time.AfterFunc(gs.config.Debounce, func() {
		if err := gs.Sync(); err != nil {
			gs.logger.Error("sync failed", "error", err)
		}
	})
}

// Sync performs the git add, commit, pull, and push operations.
func (gs *GitSync) Sync() error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.logger.Info("starting sync")

	// git add .
	if err := gs.runGit("add", "."); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// check status
	status, err := gs.runGitOutput("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	if status != "" {
		msg := fmt.Sprintf("sync: %s", time.Now().Format(time.RFC3339))
		// git commit -m msg --allow-empty (allow-empty just in case, but porcelain should catch it)
		// We use -S for auto-signing if configured in git global config
		if err := gs.runGit("commit", "-m", msg); err != nil {
			return fmt.Errorf("git commit failed: %w", err)
		}
		gs.logger.Info("committed changes")
	} else {
		gs.logger.Info("nothing to commit, checking remote")
	}

	// git pull --rebase & push if remote is configured
	if gs.config.GitRepository != "" {
		branch, err := gs.runGitOutput("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return fmt.Errorf("failed to get branch: %w", err)
		}
		branch = strings.TrimSpace(branch)

		gs.logger.Info("syncing with remote", "remote", "origin", "branch", branch)

		// Pull with rebase
		if err := gs.runGit("pull", "--rebase", "origin", branch); err != nil {
			return fmt.Errorf("git pull --rebase failed: %w (check for conflicts)", err)
		}
		gs.logger.Info("pulled from remote", "branch", branch)

		// Push
		if err := gs.runGit("push", "origin", branch); err != nil {
			return fmt.Errorf("git push failed: %w", err)
		}
		gs.logger.Info("pushed to remote", "branch", branch)
	}

	return nil
}

func (gs *GitSync) runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = gs.config.Dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s failed: %w\nOutput: %s", args[0], err, string(out))
	}
	return nil
}

func (gs *GitSync) runGitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = gs.config.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\nOutput: %s", args[0], err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}
