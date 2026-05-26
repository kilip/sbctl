package gitsync

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestIsGitInternal(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"foo/bar", false},
		{".git", true},
		{"foo/.git/bar", true},
		{"a/b/c/.git", true},
		{"a/.git/b/c", true},
	}

	for _, tt := range tests {
		result := isGitInternal(tt.path)
		if result != tt.expected {
			t.Errorf("isGitInternal(%q) = %v; expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestWatchRecursive_Error(t *testing.T) {
	gs := NewGitSync(&Config{})
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Non-existent directory should return an error
	err = gs.watchRecursive(watcher, "/nonexistentpath/foo/bar")
	if err == nil {
		t.Error("expected error for non-existent path in watchRecursive, got nil")
	}
}
