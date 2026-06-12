package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
)

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
