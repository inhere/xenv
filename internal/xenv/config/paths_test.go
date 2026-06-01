package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveDirUsesDefaultConfigDir(t *testing.T) {
	t.Setenv("XENV_CONFIG_DIR", "")
	home := t.TempDir()

	dir := ResolveDir(func() (string, error) { return home, nil })

	want := filepath.Join(home, ".config", "xenv")
	if dir != want {
		t.Fatalf("ResolveDir() = %q, want %q", dir, want)
	}
}

func TestResolveDirUsesXenvConfigDir(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "xenv-dev")
	t.Setenv("XENV_CONFIG_DIR", custom)

	dir := ResolveDir(func() (string, error) { return "ignored", nil })

	if dir != custom {
		t.Fatalf("ResolveDir() = %q, want %q", dir, custom)
	}
}

func TestResolveDirExpandsHomeInXenvConfigDir(t *testing.T) {
	t.Setenv("XENV_CONFIG_DIR", "~/.config/xenv-dev")
	home := t.TempDir()

	dir := ResolveDir(func() (string, error) { return home, nil })

	want := filepath.Join(home, ".config", "xenv-dev")
	if dir != want {
		t.Fatalf("ResolveDir() = %q, want %q", dir, want)
	}
}

func TestResolveDirFallsBackToUserConfigDirWhenHomeFails(t *testing.T) {
	t.Setenv("XENV_CONFIG_DIR", "")
	base := t.TempDir()

	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", base)
	default:
		t.Setenv("XDG_CONFIG_HOME", base)
	}

	wantBase, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir() unexpected error: %v", err)
	}

	dir := ResolveDir(func() (string, error) { return "", fmt.Errorf("boom") })
	want := filepath.Join(wantBase, "xenv")
	if dir != want {
		t.Fatalf("ResolveDir() = %q, want %q", dir, want)
	}
}

func TestDerivedPathsUseConfigDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "xenv")
	paths := PathsForDir(dir)

	checks := map[string]string{
		"config":  paths.ConfigFile,
		"global":  paths.GlobalStateFile,
		"session": paths.SessionDir,
		"index":   paths.SDKLocalIndexFile,
		"hooks":   paths.ShellHooksDir,
	}
	wants := map[string]string{
		"config":  filepath.Join(dir, "config.yaml"),
		"global":  filepath.Join(dir, "global.toml"),
		"session": filepath.Join(dir, "session"),
		"index":   filepath.Join(dir, "sdks.local.json"),
		"hooks":   filepath.Join(dir, "hooks"),
	}
	for name, got := range checks {
		if got != wants[name] {
			t.Fatalf("%s path = %q, want %q", name, got, wants[name])
		}
	}
}
