package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/config"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

func TestStateManagerAppliedDirenvRoundTrip(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)
	t.Setenv(config.EnvConfigDir, "")
	t.Setenv("XENV_HOOK_SHELL", "bash")
	xenvcom.SetHookShell("bash")
	xenvcom.SetSessionID("")
	t.Cleanup(func() {
		xenvcom.SetHookShell("")
		xenvcom.SetSessionID("")
	})

	state := NewStateManager()
	assert.Require(t, assert.NoErr(t, state.Init()))

	rec := models.NewAppliedDirenv("/proj/.xenv.toml")
	rec.AddAppliedPath("/proj/bin")
	rec.AddAppliedEnv("APP_ENV", "dev", true)
	assert.Require(t, assert.NoErr(t, state.SetAppliedDirenv(rec)))

	// 重新加载同一 session 文件时必须能读回记录
	reloaded := NewStateManager()
	assert.Require(t, assert.NoErr(t, reloaded.Init()))
	assert.Require(t, assert.NotNil(t, reloaded.AppliedDirenv()))
	assert.Eq(t, "/proj/.xenv.toml", reloaded.AppliedDirenv().File)
	assert.Eq(t, []string{"/proj/bin"}, reloaded.AppliedDirenv().Paths)

	assert.Require(t, assert.NoErr(t, reloaded.ClearAppliedDirenv()))
	assert.Nil(t, reloaded.AppliedDirenv())

	// 再次加载后记录仍然为空
	afterClear := NewStateManager()
	assert.Require(t, assert.NoErr(t, afterClear.Init()))
	assert.Nil(t, afterClear.AppliedDirenv())
}

func TestStateManagerLoadsSessionFileWithoutAppliedDirenv(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)
	t.Setenv(config.EnvConfigDir, "")
	t.Setenv("XENV_HOOK_SHELL", "bash")
	xenvcom.SetHookShell("bash")
	xenvcom.SetSessionID("")
	t.Cleanup(func() {
		xenvcom.SetHookShell("")
		xenvcom.SetSessionID("")
	})

	sessionFile := filepath.Join(config.DefaultPaths().SessionDir, xenvcom.SessionID()+".json")
	assert.Require(t, assert.NoErr(t, os.MkdirAll(filepath.Dir(sessionFile), 0o755)))
	assert.Require(t, assert.NoErr(t, os.WriteFile(sessionFile, []byte(`{"shell":"bash","sdks":{}}`), 0o644)))

	state := NewStateManager()
	assert.Require(t, assert.NoErr(t, state.Init()))
	assert.Nil(t, state.AppliedDirenv())
}

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

func TestLoadDirEnvStatePrefersLocalToml(t *testing.T) {
	projectDir := t.TempDir()
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
	projectDir := t.TempDir()
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
	projectDir := t.TempDir()
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
