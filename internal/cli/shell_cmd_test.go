package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellCommandCreatesMissingConfig(t *testing.T) {
	configDir := t.TempDir()
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./cmd/xenv", "shell", "--type", "bash")
	cmd.Dir = repoRoot
	cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+configDir, "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected shell command to auto-create config, err=%v, stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "#!/usr/bin/env bash") {
		t.Fatalf("expected shell command to print bash hook, got: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(configDir, "config.yaml")); err != nil {
		t.Fatalf("expected shell command to create config.yaml: %v", err)
	}
}

func TestShellCommandAcceptsShellExecutablePathAsType(t *testing.T) {
	configDir := t.TempDir()
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./cmd/xenv", "shell", "--type", "C:/Program Files/Git/usr/bin/bash")
	cmd.Dir = repoRoot
	cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+configDir, "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected shell command to accept executable path type, err=%v, stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "#!/usr/bin/env bash") {
		t.Fatalf("expected shell command to print bash hook, got: %s", stdout.String())
	}
}

func TestShellInstallAcceptsShellExecutablePathAsType(t *testing.T) {
	configDir := t.TempDir()
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./cmd/xenv", "shell", "--install", "--type", "C:/Program Files/Git/usr/bin/bash")
	cmd.Dir = repoRoot
	cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+configDir, "NO_COLOR=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected shell install to accept executable path type, err=%v, output=%s", err, out)
	}
}

func TestHelpDoesNotCreateConfig(t *testing.T) {
	configDir := t.TempDir()
	cmd := exec.Command("go", "run", "./cmd/xenv", "--help")
	cmd.Dir = filepath.Clean(filepath.Join("..", ".."))
	cmd.Env = appendWithoutEnv(os.Environ(), "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env, "XENV_CONFIG_DIR="+configDir, "NO_COLOR=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected help to succeed, err=%v, output=%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(configDir, "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected help not to create config.yaml, stat err=%v", err)
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
