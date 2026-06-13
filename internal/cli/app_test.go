package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/color"
	"github.com/gookit/goutil/byteutil"
	"github.com/gookit/goutil/x/assert"
)

func TestApp_showVersion(t *testing.T) {
	app := NewApp()
	app.Version = "0.0.1"

	buf := byteutil.NewBuffer()
	color.SetOutput(buf)
	defer color.ResetOutput()

	t.Run("use flag -V", func(t *testing.T) {
		code := app.RunArgs("-V")
		assert.Equal(t, 0, code)
		text := buf.ResetAndGet()
		assert.StrContainsAll(t, text, []string{app.Desc, "Version", app.Version})
	})

	t.Run("use flag --version", func(t *testing.T) {
		code := app.RunArgs("--version")
		assert.Equal(t, 0, code)
		text := buf.ResetAndGet()
		assert.StrContainsAll(t, text, []string{app.Desc, "Version", app.Version})
	})
}

func TestNewAppRegistersTopLevelCommands(t *testing.T) {
	app := NewApp()

	if app.Name != "xenv" {
		t.Fatalf("expected app name xenv, got %q", app.Name)
	}

	for _, name := range []string{
		"sdk",
		"check",
		"use",
		"unuse",
		"env",
		"path",
		"config",
		"status",
		"shell",
		"shell-init-hook",
		"shell-direnv",
	} {
		if !app.HasCommand(name) {
			t.Fatalf("expected app to register command %q", name)
		}
	}

	if app.HasCommand("tools") {
		t.Fatalf("tools command must not be registered")
	}
	if app.HasCommand("list") {
		t.Fatalf("list command must not be registered at top level; use status or sdk/env/path list")
	}
	if app.ResolveAlias("ls") == "list" {
		t.Fatalf("ls alias must not resolve to removed top-level list command")
	}
	if app.HasCommand("init") {
		t.Fatalf("init command must be registered under config, not at top level")
	}
	configCmd := app.GetCommand("config")
	if configCmd == nil {
		t.Fatal("expected config command to be registered")
	}
	if configCmd.Commands()["init"] == nil {
		t.Fatalf("expected config command to register init subcommand")
	}

	if app.ResolveAlias("init-direnv") != "shell-direnv" {
		t.Fatalf("expected init-direnv alias to resolve to shell-direnv")
	}
}

func TestEnvSetSaveDirenvFlagWritesXenvToml(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "xenv-test.exe")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/xenv")
	build.Dir = filepath.Clean(filepath.Join("..", ".."))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build xenv test binary: %v, output=%s", err, out)
	}

	tests := map[string][]string{
		"top-level set": {"set", "-s", "JAVA_TOOL_OPTIONS", "-Dfile.encoding=UTF-8"},
		"env set":       {"env", "set", "-s", "JAVA_TOOL_OPTIONS", "-Dfile.encoding=UTF-8"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			homeDir := filepath.Join(tempDir, strings.ReplaceAll(name, " ", "-"), "home")
			projectDir := filepath.Join(tempDir, strings.ReplaceAll(name, " ", "-"), "project")
			assert.Require(t, assert.NoErr(t, os.MkdirAll(homeDir, 0o755)))
			assert.Require(t, assert.NoErr(t, os.MkdirAll(projectDir, 0o755)))

			cmd := exec.Command(binPath, args...)
			cmd.Dir = projectDir
			cmd.Env = appendWithoutEnv(os.Environ(), "HOME", "USERPROFILE", "XENV_HOOK_SHELL", "XENV_SESSION_ID", "XENV_CONFIG_DIR", "NO_COLOR")
			cmd.Env = append(cmd.Env,
				"HOME="+homeDir,
				"USERPROFILE="+homeDir,
				"XENV_HOOK_SHELL=pwsh",
				"NO_COLOR=1",
			)

			out, err := cmd.CombinedOutput()
			assert.Require(t, assert.NoErr(t, err))
			output := string(out)
			assert.Contains(t, output, "Set JAVA_TOOL_OPTIONS=-Dfile.encoding=UTF-8 for direnv state")
			assert.NotContains(t, output, "for current session")

			stateFile := filepath.Join(projectDir, ".xenv.toml")
			data, err := os.ReadFile(stateFile)
			assert.Require(t, assert.NoErr(t, err))
			contents := string(data)
			assert.Contains(t, contents, "[envs]")
			assert.Contains(t, contents, `JAVA_TOOL_OPTIONS = "-Dfile.encoding=UTF-8"`)

			sessionDir := filepath.Join(homeDir, ".config", "xenv", "session")
			if entries, err := os.ReadDir(sessionDir); err == nil && len(entries) > 0 {
				var names []string
				for _, entry := range entries {
					names = append(names, entry.Name())
				}
				t.Fatalf("expected no session state files, got %s", strings.Join(names, ", "))
			}
		})
	}
}

func TestShellDirenvOutputsWarningExpressionOnInvalidDirenvState(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "xenv-test.exe")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/xenv")
	build.Dir = filepath.Clean(filepath.Join("..", ".."))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build xenv test binary: %v, output=%s", err, out)
	}

	homeDir := filepath.Join(tempDir, "home")
	projectDir := filepath.Join(tempDir, "project")
	assert.Require(t, assert.NoErr(t, os.MkdirAll(homeDir, 0o755)))
	assert.Require(t, assert.NoErr(t, os.MkdirAll(projectDir, 0o755)))
	stateFile := filepath.Join(projectDir, ".xenv.toml")
	assert.Require(t, assert.NoErr(t, os.WriteFile(stateFile, []byte("[envs\n  BAD = \"value\"\n"), 0o644)))

	cmd := exec.Command(binPath, "init-direnv")
	cmd.Dir = projectDir
	cmd.Env = appendWithoutEnv(os.Environ(), "HOME", "USERPROFILE", "XENV_HOOK_SHELL", "XENV_SESSION_ID", "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env,
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
		"XENV_HOOK_SHELL=bash",
		"NO_COLOR=1",
	)

	out, err := cmd.CombinedOutput()
	assert.Require(t, assert.NoErr(t, err))
	output := string(out)
	assert.Contains(t, output, "WARN: failed to initialize xenv direnv state")
	assert.Contains(t, output, stateFile)
	assert.Contains(t, output, "--Expression--")
	assert.True(t, strings.Index(output, "WARN:") < strings.Index(output, "--Expression--"))
}

func TestHookCommandOutputsWarningExpressionOnInvalidDirenvState(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "xenv-test.exe")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/xenv")
	build.Dir = filepath.Clean(filepath.Join("..", ".."))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build xenv test binary: %v, output=%s", err, out)
	}

	homeDir := filepath.Join(tempDir, "home")
	projectDir := filepath.Join(tempDir, "project")
	assert.Require(t, assert.NoErr(t, os.MkdirAll(homeDir, 0o755)))
	assert.Require(t, assert.NoErr(t, os.MkdirAll(projectDir, 0o755)))
	stateFile := filepath.Join(projectDir, ".xenv.toml")
	assert.Require(t, assert.NoErr(t, os.WriteFile(stateFile, []byte("[envs\n  BAD = \"value\"\n"), 0o644)))

	cmd := exec.Command(binPath, "set", "FOO", "BAR")
	cmd.Dir = projectDir
	cmd.Env = appendWithoutEnv(os.Environ(), "HOME", "USERPROFILE", "XENV_HOOK_SHELL", "XENV_SESSION_ID", "XENV_CONFIG_DIR", "NO_COLOR")
	cmd.Env = append(cmd.Env,
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
		"XENV_HOOK_SHELL=bash",
		"NO_COLOR=1",
	)

	out, err := cmd.CombinedOutput()
	assert.Require(t, assert.NoErr(t, err))
	output := string(out)
	assert.Contains(t, output, "WARN: failed to initialize xenv command")
	assert.Contains(t, output, stateFile)
	assert.Contains(t, output, "--Expression--")
	assert.NotContains(t, output, "ERROR:")
	assert.True(t, strings.Index(output, "WARN:") < strings.Index(output, "--Expression--"))
}
