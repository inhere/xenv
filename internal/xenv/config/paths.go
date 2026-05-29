package config

import (
	"os"
	"path/filepath"
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
		return filepath.Clean(dir)
	}
	home, err := homeFn()
	if err != nil || home == "" {
		return filepath.Join(".config", "xenv")
	}
	return filepath.Join(home, ".config", "xenv")
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
