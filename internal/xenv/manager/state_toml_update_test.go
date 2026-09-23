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
	state := models.NewActivityState(".xenv.toml")
	state.Paths = []string{"./bin"}
	state.ToolRequirements["rg"] = "*"

	updater := NewTomlUpdater().SetContents([]byte("paths = [\"./bin\"]\n"))
	got := string(updater.Build(state).LastContents())

	if !strings.Contains(got, "[tools]\nrg = \"*\"") {
		t.Fatalf("expected tools section in updated TOML, got:\n%s", got)
	}
}

func TestStateTomlUpdaterKeepsEmptyEnvValue(t *testing.T) {
	contents := "paths = []\n\n[envs]\nFOO = \"bar\"\n"

	state := models.NewActivityState(".xenv.toml")
	state.Envs["FOO"] = ""

	got := string(NewTomlUpdater().SetContents([]byte(contents)).Build(state).LastContents())
	assert.Contains(t, got, `FOO = ""`)

	// state 中删除该键时, 行才会被移除
	delete(state.Envs, "FOO")
	got = string(NewTomlUpdater().SetContents([]byte(contents)).Build(state).LastContents())
	assert.NotContains(t, got, "FOO")
}

func TestStateTomlUpdaterWritesNewStateWhenFileIsEmpty(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), ".xenv.toml")
	err := os.WriteFile(stateFile, nil, 0o644)
	assert.Require(t, assert.NoErr(t, err))

	state := models.NewActivityState(stateFile)
	state.SDKs["go"] = "1.24"

	err = NewTomlUpdater().Update(state)
	assert.Require(t, assert.NoErr(t, err))

	data, err := os.ReadFile(stateFile)
	assert.Require(t, assert.NoErr(t, err))
	contents := string(data)
	assert.StrContains(t, contents, "[sdks]")
	assert.StrContains(t, contents, `go = "1.24"`)
}
