package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSetAndImportPersistValues(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("bin_dir: ~/.local/bin\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	importFile := filepath.Join(configDir, "import.json")
	importData := `{"bin_dir":"D:/imported/bin","allow_up_match":0,"sdks":[{"name":"go","install_dir":"/sdk/go{version}"}]}`
	if err := os.WriteFile(importFile, []byte(importData), 0o644); err != nil {
		t.Fatalf("write import file: %v", err)
	}

	runXenv := func(args ...string) string {
		t.Helper()

		cmd := exec.Command("go", append([]string{"run", "./cmd/xenv"}, args...)...)
		cmd.Dir = filepath.Clean(filepath.Join("..", ".."))
		cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
		cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+configDir, "NO_COLOR=1")

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed, err=%v, output=%s", args, err, out)
		}
		return string(out)
	}

	runXenv("config", "set", "bin_dir", "D:/custom/bin")
	// 新进程读取时必须已是新值(即已经落盘)
	if out := runXenv("config", "get", "bin_dir"); !strings.Contains(out, "bin_dir=D:/custom/bin") {
		t.Fatalf("expected config set to persist, got %s", out)
	}

	runXenv("config", "import", importFile)
	if out := runXenv("config", "get", "bin_dir"); !strings.Contains(out, "bin_dir=D:/imported/bin") {
		t.Fatalf("expected imported bin_dir, got %s", out)
	}
	if out := runXenv("config", "get", "allow_up_match"); !strings.Contains(out, "allow_up_match=0") {
		t.Fatalf("expected imported allow_up_match, got %s", out)
	}
	if out := runXenv("config", "get", "sdks.0.name"); !strings.Contains(out, "sdks.0.name=go") {
		t.Fatalf("expected imported sdks, got %s", out)
	}
}

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
