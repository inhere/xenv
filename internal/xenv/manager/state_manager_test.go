package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

func TestEnvrcFileName(t *testing.T) {
	oldShell := xenvcom.HookShell()
	t.Cleanup(func() { xenvcom.SetHookShell(oldShell) })

	tests := map[string]string{
		"bash": ".envrc",
		"zsh":  ".envrc",
		"":     ".envrc",
		"pwsh": ".envrc.ps1",
	}

	for shellName, want := range tests {
		t.Run(shellName, func(t *testing.T) {
			xenvcom.SetHookShell(shellName)
			assert.Eq(t, want, envrcFileName())
		})
	}
}

// resolvedTempDir 返回解析软链接后的临时目录
//
// macOS 的 t.TempDir() 返回 /var/... , 而 os.Getwd() 返回解析后的 /private/var/...,
// 状态文件路径来自后者, 测试期望必须使用同一形式
func resolvedTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return resolved
	}
	return dir
}

func TestLoadDirEnvStatePrefersLocalToml(t *testing.T) {
	projectDir := resolvedTempDir(t)
	chdirForTest(t, projectDir)

	err := os.WriteFile(filepath.Join(projectDir, ".xenv.toml"), []byte("[envs]\n  SOURCE = \"shared\"\n"), 0o644)
	assert.Require(t, assert.NoErr(t, err))
	err = os.WriteFile(filepath.Join(projectDir, ".xenv.local.toml"), []byte("[envs]\n  SOURCE = \"local\"\n"), 0o644)
	assert.Require(t, assert.NoErr(t, err))

	state := NewStateManager()
	err = state.Init()
	assert.Require(t, assert.NoErr(t, err))

	assert.Eq(t, "local", state.Merged().Envs["SOURCE"])
	assert.Eq(t, filepath.Join(projectDir, ".xenv.local.toml"), state.Nearest().File)
}

func TestLoadDirEnvStateKeepsNearestDirectoryFirst(t *testing.T) {
	projectDir := resolvedTempDir(t)
	subDir := filepath.Join(projectDir, "pkg")
	err := os.MkdirAll(subDir, 0o755)
	assert.Require(t, assert.NoErr(t, err))
	chdirForTest(t, subDir)

	err = os.WriteFile(filepath.Join(projectDir, ".xenv.local.toml"), []byte("[envs]\n  SOURCE = \"parent-local\"\n"), 0o644)
	assert.Require(t, assert.NoErr(t, err))
	err = os.WriteFile(filepath.Join(subDir, ".xenv.toml"), []byte("[envs]\n  SOURCE = \"child-shared\"\n"), 0o644)
	assert.Require(t, assert.NoErr(t, err))

	state := NewStateManager()
	err = state.Init()
	assert.Require(t, assert.NoErr(t, err))

	assert.Eq(t, "child-shared", state.Merged().Envs["SOURCE"])
	assert.Eq(t, filepath.Join(subDir, ".xenv.toml"), state.Nearest().File)
}

func TestLoadDirEnvStateErrorIncludesStateFilePath(t *testing.T) {
	projectDir := resolvedTempDir(t)
	stateFile := filepath.Join(projectDir, ".xenv.toml")
	chdirForTest(t, projectDir)

	err := os.WriteFile(stateFile, []byte("[envs\n  BAD = \"value\"\n"), 0o644)
	assert.Require(t, assert.NoErr(t, err))

	state := NewStateManager()
	err = state.Init()
	assert.Require(t, assert.Err(t, err))
	assert.Contains(t, err.Error(), stateFile)
	assert.Contains(t, err.Error(), "failed to load dir state")
}

func TestSetEnvDirenvCreatesXenvToml(t *testing.T) {
	projectDir := t.TempDir()
	chdirForTest(t, projectDir)

	state := NewStateManager()
	err := state.Init()
	assert.Require(t, assert.NoErr(t, err))
	err = state.SetEnv("JAVA_TOOL_OPTIONS", "-Dfile.encoding=UTF-8", models.OpFlagDirenv)
	assert.Require(t, assert.NoErr(t, err))

	data, err := os.ReadFile(filepath.Join(projectDir, ".xenv.toml"))
	assert.Require(t, assert.NoErr(t, err))
	contents := string(data)
	assert.StrContains(t, contents, "[envs]")
	assert.StrContains(t, contents, `JAVA_TOOL_OPTIONS = "-Dfile.encoding=UTF-8"`)
}

func TestSetGlobalEnvUsesXenvConfigDir(t *testing.T) {
	rootDir := t.TempDir()
	homeDir := filepath.Join(rootDir, "home")
	configDir := filepath.Join(rootDir, "xenv-config")
	projectDir := filepath.Join(rootDir, "project")
	err := os.MkdirAll(projectDir, 0o755)
	assert.Require(t, assert.NoErr(t, err))
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("XENV_CONFIG_DIR", configDir)
	chdirForTest(t, projectDir)

	state := NewStateManager()
	err = state.Init()
	assert.Require(t, assert.NoErr(t, err))
	err = state.SetEnv("GOPROXY", "direct", models.OpFlagGlobal)
	assert.Require(t, assert.NoErr(t, err))

	data, err := os.ReadFile(filepath.Join(configDir, "global.toml"))
	assert.Require(t, assert.NoErr(t, err))
	assert.StrContains(t, string(data), `GOPROXY = "direct"`)

	_, err = os.Stat(filepath.Join(homeDir, ".config", "xenv", "global.toml"))
	assert.Require(t, assert.True(t, os.IsNotExist(err)))
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	oldWd, err := os.Getwd()
	assert.Require(t, assert.NoErr(t, err))
	err = os.Chdir(dir)
	assert.Require(t, assert.NoErr(t, err))
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
}
