package manager

import (
	"strings"
	"testing"

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
