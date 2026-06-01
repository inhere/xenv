package config

import (
	"os"
	"path/filepath"
	"strings"
)

const EnvConfigDir = "XENV_CONFIG_DIR"

type Paths struct {
	ConfigDir         string
	ConfigFile        string
	GlobalStateFile   string
	SessionDir        string
	SDKLocalIndexFile string
	ShellHooksDir     string
}

func ResolveDir(homeFn func() (string, error)) string {
	if dir := os.Getenv(EnvConfigDir); dir != "" {
		return filepath.Clean(expandHome(dir, homeFn))
	}
	home, err := homeFn()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "xenv")
	}
	if dir, cfgErr := os.UserConfigDir(); cfgErr == nil && dir != "" {
		return filepath.Join(dir, "xenv")
	}
	if abs, absErr := filepath.Abs(filepath.Join(".config", "xenv")); absErr == nil {
		return abs
	}
	return filepath.Join(os.TempDir(), "xenv")
}

func expandHome(path string, homeFn func() (string, error)) string {
	if path == "~" {
		if home, err := homeFn(); err == nil && home != "" {
			return home
		}
		return path
	}

	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		if home, err := homeFn(); err == nil && home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func PathsForDir(dir string) Paths {
	return Paths{
		ConfigDir:         dir,
		ConfigFile:        filepath.Join(dir, "config.yaml"),
		GlobalStateFile:   filepath.Join(dir, "global.toml"),
		SessionDir:        filepath.Join(dir, "session"),
		SDKLocalIndexFile: filepath.Join(dir, "sdks.local.json"),
		ShellHooksDir:     filepath.Join(dir, "hooks"),
	}
}

func DefaultPaths() Paths {
	return PathsForDir(ResolveDir(os.UserHomeDir))
}
