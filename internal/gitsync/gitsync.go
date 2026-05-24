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
	gconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitSync struct {
	config *Config
	repo   *git.Repository
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
	repo, err := git.PlainOpen(dir)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			gs.logger.Info("initializing git repository", "path", dir)
			repo, err = git.PlainInit(dir, false)
			if err != nil {
				return fmt.Errorf("failed to init git repo: %w", err)
			}
		} else {
			return fmt.Errorf("failed to open git repo: %w", err)
		}
	}
	gs.repo = repo

	// Handle remote
	if gs.config.GitRepository != "" {
		_, err = repo.Remote("origin")
		if err != nil {
			if err == git.ErrRemoteNotFound {
				gs.logger.Info("adding remote origin", "url", gs.config.GitRepository)
				_, err = repo.CreateRemote(&gconfig.RemoteConfig{
					Name: "origin",
					URLs: []string{gs.config.GitRepository},
				})
				if err != nil {
					return fmt.Errorf("failed to create remote: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get remote: %w", err)
			}
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Recursive watch
	if err := gs.watchRecursive(watcher, dir); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	gs.logger.Info("gitsync started", "path", dir, "debounce", gs.config.Debounce)

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

	name := gs.config.UserName
	if name == "" {
		name = "sbctl"
	}
	email := gs.config.UserEmail
	if email == "" {
		email = "sbctl@local"
	}

	commit, err := w.Commit(fmt.Sprintf("sync: %s", time.Now().Format(time.RFC3339)), &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	gs.logger.Info("committed changes", "hash", commit.String())

	// TODO: git pull & push if remote is configured
	if gs.config.GitRepository != "" {
		gs.logger.Info("pushing to remote", "remote", "origin")
		// Implement pull/push logic here
	}

	return nil
}
