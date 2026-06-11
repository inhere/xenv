package util

import (
	"runtime"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

func TestFormatShellPathUsesGitBashDrivePathOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git Bash path conversion is Windows-specific")
	}

	oldHookShell := xenvcom.HookShell()
	xenvcom.SetHookShell("bash")
	t.Cleanup(func() {
		xenvcom.SetHookShell(oldHookShell)
	})

	t.Run("slash path", func(t *testing.T) {
		assert.Eq(t, "/d/work/env/devsdk/gosdk/go1.24.6/bin", FormatShellPath("D:/work/env/devsdk/gosdk/go1.24.6/bin"))
	})

	t.Run("backslash path", func(t *testing.T) {
		assert.Eq(t, "/c/Users/inhere/.xenv/shims", FormatShellPath(`C:\Users\inhere\.xenv\shims`))
	})
}

func TestSplitPathUsesBashSeparatorsInGitBashHook(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git Bash path splitting is Windows-specific")
	}

	oldHookShell := xenvcom.HookShell()
	xenvcom.SetHookShell("bash")
	t.Cleanup(func() {
		xenvcom.SetHookShell(oldHookShell)
	})

	t.Run("bash style path", func(t *testing.T) {
		got := SplitPath("/c/Users/inhere/.xenv/shims:/usr/bin:/d/tools/bin")
		assert.Eq(t, []string{"/c/Users/inhere/.xenv/shims", "/usr/bin", "/d/tools/bin"}, got)
	})

	t.Run("msys converted windows path", func(t *testing.T) {
		got := SplitPath(`C:\Users\inhere\.xenv\shims;D:\tools\bin`)
		assert.Eq(t, []string{`C:\Users\inhere\.xenv\shims`, `D:\tools\bin`}, got)
	})

	t.Run("mixed path with windows drive colons", func(t *testing.T) {
		got := SplitPath(`D:/work/env/devsdk/gosdk/go1.24.6/bin:C:\Users\inhere\.xenv\shims:/usr/bin`)
		assert.Eq(t, []string{`D:/work/env/devsdk/gosdk/go1.24.6/bin`, `C:\Users\inhere\.xenv\shims`, `/usr/bin`}, got)
	})
}
