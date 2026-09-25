package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
)

func TestStateTomlUpdaterAddsToolsSection(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), ".xenv.toml")
	assert.Require(t, assert.NoErr(t, os.WriteFile(stateFile, []byte("paths = [\"./bin\"]\n"), 0o644)))

	state := models.NewActivityState(stateFile)
	state.Paths = []string{"./bin"}
	state.ToolRequirements["rg"] = "*"

	assert.NoErr(t, NewTomlUpdater().Update(state))
	got := readTomlFile(t, stateFile)

	if !strings.Contains(got, "[tools]") || !strings.Contains(got, `rg = "*"`) {
		t.Fatalf("expected tools section in updated TOML, got:\n%s", got)
	}
}

func TestStateTomlUpdaterKeepsEmptyEnvValue(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), ".xenv.toml")
	contents := "paths = []\n\n[envs]\nFOO = \"bar\"\n"
	assert.Require(t, assert.NoErr(t, os.WriteFile(stateFile, []byte(contents), 0o644)))

	state := models.NewActivityState(stateFile)
	state.Envs["FOO"] = ""

	assert.NoErr(t, NewTomlUpdater().Update(state))
	got := readTomlFile(t, stateFile)
	assert.Contains(t, got, `FOO = ""`)

	// 空值也是有效配置, 只有 state 中删除该键时, 行才会被移除
	delete(state.Envs, "FOO")
	assert.NoErr(t, NewTomlUpdater().Update(state))
	assert.NotContains(t, readTomlFile(t, stateFile), "FOO")
}

func TestStateTomlUpdaterWritesNewStateWhenFileIsEmpty(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), ".xenv.toml")
	err := os.WriteFile(stateFile, nil, 0o644)
	assert.Require(t, assert.NoErr(t, err))

	state := models.NewActivityState(stateFile)
	state.SDKs["go"] = "1.24"

	err = NewTomlUpdater().Update(state)
	assert.Require(t, assert.NoErr(t, err))

	contents := readTomlFile(t, stateFile)
	assert.StrContains(t, contents, "[sdks]")
	assert.StrContains(t, contents, `go = "1.24"`)
}
