package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigGetSupportsPathKeys(t *testing.T) {
	configDir := t.TempDir()
	configData := []byte(`
sdks:
  - name: go
    install_dir: "/sdk/go{version}"
`)
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), configData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/xenv", "config", "get", "sdks.0.name")
	cmd.Dir = filepath.Clean(filepath.Join("..", ".."))
	cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+configDir, "NO_COLOR=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected config get to succeed, err=%v, output=%s", err, out)
	}

	want := "sdks.0.name=go"
	if !strings.Contains(string(out), want) {
		t.Fatalf("expected output to contain %q, got %s", want, out)
	}
}
