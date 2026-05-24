package gitsync

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// watchRecursive adds a directory and all its subdirectories to the watcher.
func (gs *GitSync) watchRecursive(watcher *fsnotify.Watcher, path string) error {
	return filepath.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if isGitInternal(newPath) {
				return filepath.SkipDir
			}
			gs.logger.Debug("watching directory", "path", newPath)
			return watcher.Add(newPath)
		}
		return nil
	})
}

// isGitInternal checks if a path is inside the .git directory.
func isGitInternal(path string) bool {
	base := filepath.Base(path)
	if base == ".git" {
		return true
	}
	// Check if any parent is .git
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if part == ".git" {
			return true
		}
	}
	return false
}
