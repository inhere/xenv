package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
)

func TestListActivityHidesEmptySections(t *testing.T) {
	state := models.NewActivityState("global.toml")
	state.SDKs["go"] = "1.21.13"

	out := strings.Join(formatActivityLines(state), "\n")

	assert.Contains(t, out, "Active Develop SDKs:")
	assert.Contains(t, out, "go")
	assert.Contains(t, out, "1.21.13")
	assert.NotContains(t, out, "Active Env Variables:")
	assert.NotContains(t, out, "Active PATH Entries:")
	assert.NotContains(t, out, "Tool Requirements:")
}

func TestListStateGroupShowsSourceFile(t *testing.T) {
	state := models.NewActivityState("global.toml")
	state.SDKs["go"] = "1.21.13"

	out := captureListOutput(t, func() {
		listStateGroup("Global State", state, "No global state found")
	})

	assert.Contains(t, out, " - from: global.toml")
	assert.NotContains(t, out, "No global state found")
}

func captureListOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	assert.Require(t, assert.NoErr(t, err))
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	assert.Require(t, assert.NoErr(t, writer.Close()))
	stdoutBytes, err := io.ReadAll(reader)
	assert.Require(t, assert.NoErr(t, err))

	output := string(stdoutBytes)
	return strings.ReplaceAll(output, "\r\n", "\n")
}
