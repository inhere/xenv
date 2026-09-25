package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
)

const tomlUpdateFixture = `# my project env
paths = [
  "./bin",
]

[sdks]
# pinned
go = "1.22"
node = "20"

[envs]

[tools]

[custom_section]
# not managed by xenv
note = "keep me"
`

func writeTomlFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".xenv.toml")
	assert.NoErr(t, os.WriteFile(path, []byte(tomlUpdateFixture), 0o644))
	return path
}

func readTomlFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoErr(t, err)
	return string(data)
}

func TestTomlUpdaterKeepsUntouchedContent(t *testing.T) {
	path := writeTomlFixture(t)

	state := models.NewActivityState(path)
	state.Paths = []string{"./bin"}
	state.SDKs["go"] = "1.23"
	state.SDKs["node"] = "20"
	state.Envs["FOO"] = "bar"

	assert.NoErr(t, NewTomlUpdater().Update(state))
	got := readTomlFile(t, path)

	// The header comment and the array layout survive: paths did not change.
	assert.StrContains(t, got, "# my project env")
	assert.StrContains(t, got, "paths = [\n  \"./bin\",\n]")
	// The changed key keeps its own comment, the untouched one keeps its line.
	assert.StrContains(t, got, "# pinned")
	assert.StrContains(t, got, `go = "1.23"`)
	assert.StrContains(t, got, "\nnode = \"20\"\n")
	// A new key lands in its section.
	assert.StrContains(t, got, `FOO = "bar"`)
	// A section xenv does not manage is left alone, comment included.
	assert.StrContains(t, got, "[custom_section]")
	assert.StrContains(t, got, "# not managed by xenv")
	assert.StrContains(t, got, `note = "keep me"`)
}

func TestTomlUpdaterAppliesRemovalsAndNewSections(t *testing.T) {
	path := writeTomlFixture(t)

	state := models.NewActivityState(path)
	state.ToolRequirements["rg"] = "*"

	assert.NoErr(t, NewTomlUpdater().Update(state))
	got := readTomlFile(t, path)

	// Every sdks entry left the state, so those lines are gone.
	assert.False(t, strings.Contains(got, "go = "))
	assert.False(t, strings.Contains(got, "node = "))
	// A section that was empty gains the new requirement.
	assert.StrContains(t, got, "[tools]")
	assert.StrContains(t, got, `rg = "*"`)
	// Sections and comments that are not part of the model stay.
	assert.StrContains(t, got, `note = "keep me"`)
	// paths is now empty in the state, so the value follows.
	assert.StrContains(t, got, "paths = []")
}
