package sysenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestRcBlockUpsertCreatesManagedBlock(t *testing.T) {
	filePath := writeRcTestFile(t, "export EDITOR=vim\n")

	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcEnvLine("FOO"), rcEnvLine("FOO", "bar"))))

	assert.Eq(t, "export EDITOR=vim\n"+
		"\n"+
		"# >>> xenv system env >>>\n"+
		"export FOO='bar'\n"+
		"# <<< xenv system env <<<\n", readRcTestFile(t, filePath))
}

func TestRcBlockUpsertReplacesSameKey(t *testing.T) {
	filePath := writeRcTestFile(t, "")
	match := matchRcEnvLine("FOO")

	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, match, rcEnvLine("FOO", "one"))))
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, match, rcEnvLine("FOO", "two"))))
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcPathLine("/opt/bin"), rcPathLine("/opt/bin"))))

	content := readRcTestFile(t, filePath)
	assert.Eq(t, "# >>> xenv system env >>>\n"+
		"export FOO='two'\n"+
		"export PATH='/opt/bin':$PATH\n"+
		"# <<< xenv system env <<<\n", content)
}

func TestRcBlockRemoveKeepsOtherEntries(t *testing.T) {
	filePath := writeRcTestFile(t, "")
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcEnvLine("FOO"), rcEnvLine("FOO", "bar"))))
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcEnvLine("BAZ"), rcEnvLine("BAZ", "qux"))))

	found, err := removeBlockLine(filePath, matchRcEnvLine("FOO"))
	assert.Require(t, assert.NoErr(t, err))
	assert.True(t, found)

	assert.Eq(t, "# >>> xenv system env >>>\n"+
		"export BAZ='qux'\n"+
		"# <<< xenv system env <<<\n", readRcTestFile(t, filePath))

	// 删除不存在的条目时保持文件不变
	found, err = removeBlockLine(filePath, matchRcEnvLine("FOO"))
	assert.Require(t, assert.NoErr(t, err))
	assert.True(t, !found)
}

func TestRcBlockRemovedAfterLastEntryRemoved(t *testing.T) {
	filePath := writeRcTestFile(t, "export EDITOR=vim\n")
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcPathLine("/opt/bin"), rcPathLine("/opt/bin"))))

	found, err := removeBlockLine(filePath, matchRcPathLine("/opt/bin"))
	assert.Require(t, assert.NoErr(t, err))
	assert.True(t, found)
	assert.Eq(t, "export EDITOR=vim\n", readRcTestFile(t, filePath))

	// 文件不存在时按空文件处理
	missing := filepath.Join(t.TempDir(), ".zshrc")
	found, err = removeBlockLine(missing, matchRcPathLine("/opt/bin"))
	assert.Require(t, assert.NoErr(t, err))
	assert.True(t, !found)
	assert.Require(t, assert.NoErr(t, upsertBlockLine(missing, matchRcPathLine("/opt/bin"), rcPathLine("/opt/bin"))))
	assert.Eq(t, "# >>> xenv system env >>>\n"+
		"export PATH='/opt/bin':$PATH\n"+
		"# <<< xenv system env <<<\n", readRcTestFile(t, missing))
}

func TestRcPathLineRoundTrip(t *testing.T) {
	for _, path := range []string{
		"/opt/bin",
		"/opt/my tools/bin",
		"/opt/it's/bin",
	} {
		t.Run(path, func(t *testing.T) {
			value, ok := rcPathValue(rcPathLine(path))
			assert.True(t, ok)
			assert.Eq(t, path, value)
		})
	}

	// 环境变量行不是 PATH 条目行
	_, ok := rcPathValue(rcEnvLine("PATH", "/opt/bin"))
	assert.True(t, !ok)
}

func TestRcFileFor(t *testing.T) {
	homeDir := filepath.FromSlash("/home/tester")
	tests := map[string]string{
		"zsh":           ".zshrc",
		"/bin/zsh":      ".zshrc",
		"bash":          ".bashrc",
		"/usr/bin/bash": ".bashrc",
		"sh":            ".profile",
		"dash":          ".profile",
		"":              ".profile",
	}

	for shellName, want := range tests {
		t.Run(shellName, func(t *testing.T) {
			filePath, err := rcFileFor(shellName, homeDir)
			assert.Require(t, assert.NoErr(t, err))
			assert.Eq(t, filepath.Join(homeDir, want), filePath)
		})
	}

	_, err := rcFileFor("pwsh", homeDir)
	assert.ErrMsgContains(t, err, "not supported")
}

func TestRcBlockEnvVarsAndPaths(t *testing.T) {
	filePath := writeRcTestFile(t, "export EDITOR=vim\n")
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcEnvLine("FOO"), rcEnvLine("FOO", "a b"))))
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcEnvLine("BAZ"), rcEnvLine("BAZ", "qux"))))
	assert.Require(t, assert.NoErr(t, upsertBlockLine(filePath, matchRcPathLine("/opt/bin"), rcPathLine("/opt/bin"))))

	lines, err := blockContent(filePath)
	assert.Require(t, assert.NoErr(t, err))
	assert.Eq(t, map[string]string{"FOO": "a b", "BAZ": "qux"}, blockEnvVars(lines))
	assert.Eq(t, []string{"/opt/bin"}, blockPaths(lines))

	// 没有托管块的文件
	missing := filepath.Join(t.TempDir(), ".zshrc")
	lines, err = blockContent(missing)
	assert.Require(t, assert.NoErr(t, err))
	assert.Eq(t, 0, len(lines))
	assert.Eq(t, map[string]string{}, blockEnvVars(lines))
}

func TestRcEnvEntry(t *testing.T) {
	tests := map[string]struct {
		name  string
		value string
		ok    bool
	}{
		"export FOO='bar'":             {name: "FOO", value: "bar", ok: true},
		"export FOO='it'\\''s'":        {name: "FOO", value: "it's", ok: true},
		"export FOO=":                  {ok: false},
		"export FOO=bar":               {ok: false},
		"export PATH='/opt/bin':$PATH": {ok: false},
		"FOO='bar'":                    {ok: false},
	}

	for line, want := range tests {
		t.Run(line, func(t *testing.T) {
			name, value, ok := rcEnvEntry(line)
			assert.Eq(t, want.ok, ok)
			assert.Eq(t, want.name, name)
			assert.Eq(t, want.value, value)
		})
	}
}

func writeRcTestFile(t *testing.T, content string) string {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), ".bashrc")
	assert.Require(t, assert.NoErr(t, os.WriteFile(filePath, []byte(content), 0o644)))
	return filePath
}

func readRcTestFile(t *testing.T, filePath string) string {
	t.Helper()

	data, err := os.ReadFile(filePath)
	assert.Require(t, assert.NoErr(t, err))
	return string(data)
}
