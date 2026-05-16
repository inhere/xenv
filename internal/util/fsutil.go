package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/fsutil"
	"github.com/inhere/xenv/pkg/xenv/xenvcom"
)

// NormalizePath normalizes a path by expanding home directory and cleaning it.
func NormalizePath(path string) string {
	fmtPath := filepath.Clean(fsutil.ExpandPath(path))
	if xenvcom.IsHookBash() {
		fmtPath = fsutil.UnixPath(fmtPath)
	}
	return fmtPath
}

// SplitPath splits a PATH string into individual paths.
func SplitPath(envPath string) []string {
	return strings.Split(envPath, string(os.PathListSeparator))
}

// JoinPaths joins multiple path entries into a single PATH string.
func JoinPaths(paths []string) string {
	return strings.Join(paths, xenvcom.PathSep())
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return fsutil.EnsureDir(NormalizePath(path))
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

// CreateSymlink creates a symbolic link.
func CreateSymlink(target, linkPath string) error {
	exists, err := FileExists(linkPath)
	if err != nil {
		return err
	}

	if exists {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	return os.Symlink(target, linkPath)
}
