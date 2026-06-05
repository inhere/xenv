package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inhere/xenv/internal/xenv/models"
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

func TestManagerGetValueSupportsPathKeys(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := []byte(`
bin_dir: "~/.local/bin"
global_env:
  GOPROXY: "https://proxy.golang.org,direct"
sdks:
  - name: go
    install_dir: "/sdk/go{version}"
    active_env:
      GOROOT: "{install_dir}"
`)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	manager := NewConfigManager()
	if err := manager.LoadConfig(configPath); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	tests := map[string]any{
		"bin_dir":                  "~/.local/bin",
		"global_env.GOPROXY":       "https://proxy.golang.org,direct",
		"sdks.0.name":              "go",
		"sdks.0.active_env.GOROOT": "{install_dir}",
	}
	for key, want := range tests {
		got, ok := manager.GetValue(key)
		if !ok {
			t.Fatalf("GetValue(%q) not found", key)
		}
		if got != want {
			t.Fatalf("GetValue(%q) = %#v, want %#v", key, got, want)
		}
	}

	if _, ok := manager.GetValue("sdks.1.name"); ok {
		t.Fatal("expected missing path key to return ok=false")
	}
}

func TestFindSDKConfigSupportsAlias(t *testing.T) {
	cfg := &models.Configuration{
		SDKs: []models.ToolChain{{
			Name:  "jdk",
			Alias: "java",
		}},
	}

	got := cfg.FindSDKConfig("java")
	if got == nil {
		t.Fatal("expected alias java to resolve jdk config")
	}
	if got.Name != "jdk" {
		t.Fatalf("FindSDKConfig(java).Name = %q, want jdk", got.Name)
	}
}
