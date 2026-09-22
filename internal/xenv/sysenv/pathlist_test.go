package sysenv

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestSplitPathListSkipsBlankEntries(t *testing.T) {
	assert.Eq(t, []string{`C:\bin`, `D:\tools`}, splitPathList(` C:\bin ;; D:\tools ; `))
	assert.Eq(t, []string(nil), splitPathList(""))
}

func TestPathListIndexIgnoresCaseAndSeparators(t *testing.T) {
	pathList := []string{`%USERPROFILE%\bin`, `C:\Program Files\Git\cmd`, `D:\env\bin`}

	assert.Eq(t, 2, pathListIndex(pathList, `d:/env/bin/`))
	assert.Eq(t, 1, pathListIndex(pathList, `c:\program files\git\cmd`))
	assert.Eq(t, 0, pathListIndex(pathList, `%USERPROFILE%\bin`))
	assert.Eq(t, -1, pathListIndex(pathList, `D:\env\sbin`))
}
