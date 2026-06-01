package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellCommandConfigErrorDoesNotPrintToStdout(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./cmd/xenv", "shell", "--type", "bash")
	cmd.Dir = repoRoot
	cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+t.TempDir(), "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected shell command to fail when config file is missing, stdout: %s", stdout.String())
	}
	if stdout.Len() > 0 {
		t.Fatalf("expected config error not to be printed to stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "failed to initialize configuration") {
		t.Fatalf("expected config error on stderr, got: %s", stderr.String())
	}
}

func TestConfigInitCreatesMissingConfig(t *testing.T) {
	configDir := t.TempDir()
	cmd := exec.Command("go", "run", "./cmd/xenv", "cfg", "init")
	cmd.Dir = filepath.Clean(filepath.Join("..", ".."))
	cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+configDir, "NO_COLOR=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected config init to create missing config, err=%v, output=%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(configDir, "config.yaml")); err != nil {
		t.Fatalf("expected config init to create config.yaml: %v", err)
	}
}

func appendWithoutEnv(env []string, names ...string) []string {
	removed := make(map[string]struct{}, len(names))
	for _, name := range names {
		removed[name+"="] = struct{}{}
	}

	filtered := env[:0]
	for _, item := range env {
		keep := true
		for prefix := range removed {
			if strings.HasPrefix(item, prefix) {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
