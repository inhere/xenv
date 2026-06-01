package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigExpandsEnvValues(t *testing.T) {
	t.Setenv("XENV_TEST_ROOT", "/opt/xenv")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := []byte(`bin_dir: "${XENV_TEST_ROOT}/bin"`)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	manager := NewConfigManager()
	if err := manager.LoadConfig(configPath); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := "/opt/xenv/bin"
	if manager.Config.BinDir != want {
		t.Fatalf("BinDir = %q, want %q", manager.Config.BinDir, want)
	}
}
