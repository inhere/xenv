package util

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/gookit/goutil/fsutil"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

// NormalizePath normalizes a path by expanding home directory and cleaning it.
func NormalizePath(path string) string {
	return filepath.Clean(fsutil.ExpandPath(path))
}

// FormatShellPath formats a filesystem path for the active shell script output.
func FormatShellPath(path string) string {
	fmtPath := NormalizePath(path)
	if xenvcom.IsHookBash() {
		return toGitBashPath(fsutil.UnixPath(fmtPath))
	}
	return fmtPath
}

// FormatShellPathFor formats a filesystem path for scripts generated for shell.
func FormatShellPathFor(path, shell string) string {
	fmtPath := NormalizePath(path)
	if runtime.GOOS == "windows" && shell == "bash" {
		return toGitBashPath(fsutil.UnixPath(fmtPath))
	}
	return fmtPath
}

var winDrivePathRe = regexp.MustCompile(`^([A-Za-z]):(/.*)?$`)

func toGitBashPath(path string) string {
	matches := winDrivePathRe.FindStringSubmatch(path)
	if len(matches) == 0 {
		return path
	}

	drive := strings.ToLower(matches[1])
	rest := matches[2]
	if rest == "" {
		return "/" + drive
	}
	return "/" + drive + rest
}

// SplitPath splits a PATH string into individual paths.
func SplitPath(envPath string) []string {
	if xenvcom.IsHookBash() {
		if strings.Contains(envPath, ";") {
			return strings.Split(envPath, ";")
		}
		return splitGitBashPath(envPath)
	}
	return strings.Split(envPath, string(os.PathListSeparator))
}

func splitGitBashPath(envPath string) []string {
	var paths []string
	start := 0
	for i := 0; i < len(envPath); i++ {
		if envPath[i] != ':' {
			continue
		}
		if isWindowsDriveColon(envPath, start, i) {
			continue
		}

		paths = append(paths, envPath[start:i])
		start = i + 1
	}
	return append(paths, envPath[start:])
}

func isWindowsDriveColon(s string, start, colon int) bool {
	return colon == start+1 &&
		((s[start] >= 'A' && s[start] <= 'Z') || (s[start] >= 'a' && s[start] <= 'z')) &&
		colon+1 < len(s) &&
		(s[colon+1] == '/' || s[colon+1] == '\\')
}

// JoinPaths joins multiple path entries into a single PATH string.
func JoinPaths(paths []string) string {
	if !xenvcom.IsHookBash() {
		return strings.Join(paths, xenvcom.PathSep())
	}

	fmtPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		fmtPaths = append(fmtPaths, FormatShellPath(path))
	}
	return strings.Join(fmtPaths, xenvcom.PathSep())
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return fsutil.EnsureDir(NormalizePath(path))
}

// FileExists checks if a file exists and is not a directory.
// CopyFile copies a file from src to dst.
// CreateSymlink creates a symbolic link.
