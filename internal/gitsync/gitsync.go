package gitsync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kilip/sbctl/internal/config"
)

type GitSync struct {
	config *config.Config
	repo   *git.Repository
	mu     sync.Mutex
	timer  *time.Timer
	logger *slog.Logger
}

var (
	instance *GitSync
	once     sync.Once
)

// GetGitSync returns the singleton GitSync instance.
func GetGitSync() *GitSync {
	once.Do(func() {
		cfg := config.GetConfig()
		instance = &GitSync{
			config: cfg,
			logger: slog.Default().With("module", "gitsync"),
		}
	})
	return instance
}

// Start begins watching the vault directory for changes.
func (gs *GitSync) Start(ctx context.Context) error {
	if !gs.config.GitSync.Enabled {
		gs.logger.Info("gitsync disabled")
		return nil
	}

	vaultDir := gs.config.Vault.Dir
	if vaultDir == "" {
		return fmt.Errorf("vault directory not configured")
	}

	// Ensure it's a git repo
	repo, err := git.PlainOpen(vaultDir)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			gs.logger.Info("initializing git repository in vault directory", "path", vaultDir)
			repo, err = git.PlainInit(vaultDir, false)
			if err != nil {
				return fmt.Errorf("failed to init git repo: %w", err)
			}
		} else {
			return fmt.Errorf("failed to open git repo: %w", err)
		}
	}
	gs.repo = repo

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Recursive watch
	if err := gs.watchRecursive(watcher, vaultDir); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	gs.logger.Info("gitsync started", "path", vaultDir, "debounce", gs.config.GitSync.Debounce)

	go gs.watchLoop(ctx, watcher)

	return nil
}

func (gs *GitSync) watchLoop(ctx context.Context, watcher *fsnotify.Watcher) {
	defer watcher.Close()

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

	gs.timer = time.AfterFunc(gs.config.GitSync.Debounce, func() {
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

	w, err := gs.repo.Worktree()
	if err != nil {
		return err
	}

	// git add .
	if err := w.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// git commit
	status, err := w.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		gs.logger.Info("nothing to commit, sync skipped")
		return nil
	}

	commit, err := w.Commit(fmt.Sprintf("vault sync: %s", time.Now().Format(time.RFC3339)), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "sbctl",
			Email: "sbctl@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	gs.logger.Info("committed changes", "hash", commit.String())

	// TODO: git pull & push if remote is configured
	if gs.config.GitSync.Remote != "" {
		gs.logger.Info("pushing to remote", "remote", gs.config.GitSync.Remote)
		// Implement pull/push logic here
	}

	return nil
}
