package models

import (
	"encoding/json"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestAppliedDirenvRecord(t *testing.T) {
	rec := NewAppliedDirenv("/proj/.xenv.toml")

	assert.True(t, rec.AddAppliedPath("/proj/bin"))
	assert.True(t, !rec.AddAppliedPath("/proj/bin"), "duplicate path must be ignored")

	rec.AddAppliedEnv("APP_ENV", "dev", true)
	rec.AddAppliedEnv("APP_ENV", "other", true) // 同名保留最早的旧值
	rec.AddAppliedEnv("GOROOT", "", false)
	rec.AddAppliedSDK("go", "1.24.13")

	assert.Eq(t, []string{"/proj/bin"}, rec.Paths)
	assert.Eq(t, []AppliedEnv{
		{Name: "APP_ENV", Prev: "dev", HadPrev: true},
		{Name: "GOROOT", Prev: "", HadPrev: false},
	}, rec.Envs)
	assert.Eq(t, map[string]string{"go": "1.24.13"}, rec.SDKs)
	assert.True(t, !rec.IsEmpty())

	assert.True(t, rec.RemoveAppliedEnv("APP_ENV"))
	assert.True(t, !rec.RemoveAppliedEnv("APP_ENV"))
	assert.Eq(t, 1, len(rec.Envs))
}

func TestActivityStateAppliedDirenvIsNotContent(t *testing.T) {
	state := NewActivityState("session.json")
	state.Shell = "bash"

	// 记录不参与 IsEmpty 判断
	state.SetAppliedDirenv(NewAppliedDirenv("/proj/.xenv.toml"))
	assert.True(t, state.IsEmpty())
	assert.True(t, state.HasUpdate)
	assert.True(t, state.HasAppliedDirenv())

	// JSON 往返保持记录内容
	data, err := json.Marshal(state)
	assert.Require(t, assert.NoErr(t, err))

	var decoded ActivityState
	assert.Require(t, assert.NoErr(t, json.Unmarshal(data, &decoded)))
	assert.Require(t, assert.True(t, decoded.HasAppliedDirenv()))
	assert.Eq(t, "/proj/.xenv.toml", decoded.AppliedDirenv.File)

	state.ClearAppliedDirenv()
	assert.True(t, !state.HasAppliedDirenv())
	assert.Eq(t, "null", string(mustJSON(t, state.AppliedDirenv)))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	assert.Require(t, assert.NoErr(t, err))
	return data
}
