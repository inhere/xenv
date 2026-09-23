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

func TestAppliedDirenvJSONRoundTrip(t *testing.T) {
	rec := NewAppliedDirenv("/proj/.xenv.toml")
	rec.AddAppliedPath("/proj/bin")
	rec.AddAppliedEnv("APP_ENV", "dev", true)
	rec.AddAppliedSDK("go", "1.24.13")

	data, err := json.Marshal(rec)
	assert.Require(t, assert.NoErr(t, err))

	var decoded AppliedDirenv
	assert.Require(t, assert.NoErr(t, json.Unmarshal(data, &decoded)))

	assert.Eq(t, rec.File, decoded.File)
	assert.Eq(t, rec.Paths, decoded.Paths)
	assert.Eq(t, rec.Envs, decoded.Envs)
	assert.Eq(t, rec.SDKs, decoded.SDKs)
	assert.True(t, !decoded.IsEmpty())

	empty := NewAppliedDirenv("/proj/.xenv.toml")
	assert.True(t, empty.IsEmpty())
}
