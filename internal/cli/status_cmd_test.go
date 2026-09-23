package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
)

func TestBuildRuntimeSDKRows(t *testing.T) {
	installDir := filepath.Join(string(filepath.Separator), "tools", "go", "1.24.0")
	binDir := filepath.Join(installDir, "bin")

	sdks := []models.InstalledSDK{
		{
			Name:       "go",
			Version:    "1.24.0",
			InstallDir: installDir,
			Config:     &models.ToolChain{Name: "go", BinDir: "bin"},
		},
		{
			Name:       "node",
			Version:    "22.0.0",
			InstallDir: filepath.Join(string(filepath.Separator), "tools", "node", "22.0.0"),
			Config:     &models.ToolChain{Name: "node", BinDir: "bin"},
		},
	}

	sep := string(os.PathListSeparator)
	pathValue := strings.Join([]string{"/other/bin", binDir, "/more/bin"}, sep)

	rows := buildRuntimeSDKRows(sdks, pathValue)
	assert.Require(t, assert.Eq(t, 1, len(rows)))
	assert.Contains(t, rows[0], "go")
	assert.Contains(t, rows[0], "1.24.0")
	assert.Contains(t, rows[0], "PATH #2")

	// PATH 中没有 SDK bin 目录时不输出
	assert.Eq(t, 0, len(buildRuntimeSDKRows(sdks, "/other/bin")))
}

func TestBuildEffectiveSDKRows(t *testing.T) {
	global := models.NewActivityState("global.toml")
	global.SDKs["go"] = "1.21.13"
	global.SDKs["java"] = "17.0.11"

	session := models.NewActivityState("session.json")
	session.SDKs["go"] = "1.24.6"
	session.SDKs["node"] = "22.0.0"

	dir := models.NewActivityState(".xenv.toml")
	dir.SDKs["go"] = "1.25"

	rows := buildEffectiveSDKRows(global, session, []*models.ActivityState{dir})

	assert.Require(t, assert.Eq(t, 3, len(rows)))

	t.Run("directory overrides session and global", func(t *testing.T) {
		assert.Eq(t, "go", rows[0].Name)
		assert.Eq(t, "1.25", rows[0].Version)
		assert.Eq(t, sourceDirectory, rows[0].Source)
		assert.Eq(t, []stateValue{
			{Source: sourceSession, Version: "1.24.6"},
			{Source: sourceGlobal, Version: "1.21.13"},
		}, rows[0].Overrides)
	})

	t.Run("global remains effective when no higher value exists", func(t *testing.T) {
		assert.Eq(t, "java", rows[1].Name)
		assert.Eq(t, "17.0.11", rows[1].Version)
		assert.Eq(t, sourceGlobal, rows[1].Source)
	})

	t.Run("session remains effective when no directory value exists", func(t *testing.T) {
		assert.Eq(t, "node", rows[2].Name)
		assert.Eq(t, "22.0.0", rows[2].Version)
		assert.Eq(t, sourceSession, rows[2].Source)
	})
}

func TestFormatEffectiveSDKRows(t *testing.T) {
	rows := []effectiveSDKRow{
		{
			Name:    "go",
			Version: "1.25",
			Source:  sourceDirectory,
			Overrides: []stateValue{
				{Source: sourceSession, Version: "1.24.6"},
				{Source: sourceGlobal, Version: "1.21.13"},
			},
		},
		{
			Name:    "node",
			Version: "22.0.0",
			Source:  sourceSession,
		},
	}

	lines := formatEffectiveSDKRows(rows)

	assert.Eq(t, []string{
		"SDKs:",
		"          go => 1.25  (Directory State)",
		"        node => 22.0.0  (Session Context)",
		"",
		"Overrides:",
		"          go: Session Context 1.24.6, Global State 1.21.13",
	}, lines)
}

func TestFormatAppliedRecordLines(t *testing.T) {
	assert.Eq(t, []string{"No applied direnv record found"}, formatAppliedRecordLines(nil))
	assert.Eq(t, []string{"No applied direnv record found"},
		formatAppliedRecordLines(models.NewAppliedDirenv("/proj/.xenv.toml")))

	rec := models.NewAppliedDirenv("/proj/.xenv.toml")
	rec.AddAppliedPath("/proj/bin")
	rec.AddAppliedEnv("APP_ENV", "dev", true)
	rec.AddAppliedEnv("NEW_ENV", "", false)
	rec.AddAppliedSDK("go", "1.24.13")

	lines := formatAppliedRecordLines(rec)
	assert.Contains(t, lines, " - from: /proj/.xenv.toml")
	assert.Contains(t, lines, "Applied PATH:")
	assert.Contains(t, lines, "  <green>1</>. /proj/bin")
	assert.Contains(t, lines, "  <green>APP_ENV</> (prev: dev)")
	assert.Contains(t, lines, "  <green>NEW_ENV</> (prev: unset)")
	assert.Contains(t, lines, "  <green>        go</> => 1.24.13")
}

func TestFormatSessionContextLines(t *testing.T) {
	session := models.NewActivityState("session.json")
	session.SDKs["go"] = "1.24.6"
	session.SDKs["node"] = "22.0.0"

	lines := formatStateSDKLines("Session Defaults:", session, map[string]string{
		"go": sourceDirectory,
	})

	assert.Contains(t, lines, "Session Defaults:")
	assert.Contains(t, lines, "  <green>        go</> => 1.24.6  (overridden by Directory State)")
	assert.Contains(t, lines, "  <green>      node</> => 22.0.0")
}
