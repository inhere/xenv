package cli

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
)

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
