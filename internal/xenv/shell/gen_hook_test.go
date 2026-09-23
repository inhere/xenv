package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

func TestGeneratedHooksBypassXenvWrapper(t *testing.T) {
	oldBinCommand := xenvcom.BinCommand
	oldBinName := xenvcom.BinName
	xenvcom.SetBinCommand("xenv")
	xenvcom.SetBinName("xenv")
	t.Cleanup(func() {
		xenvcom.BinCommand = oldBinCommand
		xenvcom.BinName = oldBinName
	})

	params := &models.GenInitScriptParams{ShellHooksDir: "~/.config/xenv/hooks"}

	t.Run("bash", func(t *testing.T) {
		script, err := NewScriptGenerator(Bash).GenHookScripts(params)
		if err != nil {
			t.Fatal(err)
		}

		assertNotContains(t, script, `export XENV_SESSION_ID=`)
		assertNotContains(t, script, `local result="$(xenv "$command" "$@")"`)
		assertNotContains(t, script, `local result="$(xenv env "$command" "$@")"`)
		assertContains(t, script, `local result="$(command xenv "$command" "$@")"`)
		assertContains(t, script, `local result="$(command xenv env "$command" "$@")"`)
	})

	t.Run("zsh", func(t *testing.T) {
		script, err := NewScriptGenerator(Zsh).GenHookScripts(params)
		if err != nil {
			t.Fatal(err)
		}

		assertNotContains(t, script, `export XENV_SESSION_ID=`)
		assertNotContains(t, script, `local result="$(xenv "$command" "$@")"`)
		assertNotContains(t, script, `local result="$(xenv env "$command" "$@")"`)
		assertContains(t, script, `local result="$(command xenv "$command" "$@")"`)
		assertContains(t, script, `local result="$(command xenv env "$command" "$@")"`)
	})

	t.Run("pwsh", func(t *testing.T) {
		script, err := NewScriptGenerator(Pwsh).GenHookScripts(params)
		if err != nil {
			t.Fatal(err)
		}

		assertNotContains(t, script, `$env:XENV_SESSION_ID =`)
		assertContains(t, script, `$script:XenvBinCommand = (Get-Command xenv -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source`)
		assertNotContains(t, script, `& xenv $Command @Arguments`)
		assertNotContains(t, script, `& xenv env $Command @Arguments`)
		assertNotContains(t, script, `& xenv shell-init-hook --type pwsh`)
		assertContains(t, script, `& $script:XenvBinCommand $Command @Arguments`)
		assertContains(t, script, `& $script:XenvBinCommand env $Command @Arguments`)
	})
}

func TestGeneratedHooksSupportProjectScripts(t *testing.T) {
	params := &models.GenInitScriptParams{
		ShellHooksDir:        "~/.config/xenv/hooks",
		SourceProjectScripts: true,
	}

	bash, err := NewScriptGenerator(Bash).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, bash, ".xenv.sh")

	zsh, err := NewScriptGenerator(Zsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, zsh, ".xenv.sh")
	assertContains(t, zsh, "invoke_xenv_result")
	assertContains(t, zsh, `local result="$(command xenv init-direnv)"`)

	pwsh, err := NewScriptGenerator(Pwsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, pwsh, ".xenv.ps1")
}

func TestGeneratedBashHookOnlyPrintsExprPartInDebugMode(t *testing.T) {
	params := &models.GenInitScriptParams{ShellHooksDir: "~/.config/xenv/hooks"}
	script, err := NewScriptGenerator(Bash).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}

	assertNotContains(t, script, "                    echo \"expr_part: $expr_part\"\n")
	assertContains(t, script, `[ "$XENV_DEBUG_MODE" = "true" ] && echo "expr_part: $expr_part"`)
}

func TestGeneratedBashHookDoesNotEnableGlobalExitOnError(t *testing.T) {
	script, err := NewScriptGenerator(Bash).GenHookScripts(&models.GenInitScriptParams{ShellHooksDir: "~/.config/xenv/hooks"})
	if err != nil {
		t.Fatal(err)
	}

	assertNotContains(t, script, "\nset -e\n")
}

func TestGeneratedHooksEvaluateCommandAliases(t *testing.T) {
	params := &models.GenInitScriptParams{ShellHooksDir: "~/.config/xenv/hooks"}

	bash, err := NewScriptGenerator(Bash).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, bash, "use|u|unuse|un|env|e|path|p)")
	assertContains(t, bash, `complete -W "use u unuse un env e set unset path p status st help" xenv`)
	assertNotContains(t, bash, ` path p list help`)

	zsh, err := NewScriptGenerator(Zsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, zsh, "use|u|unuse|un|env|e|path|p)")
	assertContains(t, zsh, `compctl -k "use u unuse un env e set unset path p status st help" xenv`)
	assertNotContains(t, zsh, ` path p list help`)

	pwsh, err := NewScriptGenerator(Pwsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, pwsh, "{ $_ -in @('use', 'u', 'unuse', 'un', 'env', 'e', 'path', 'p') }")
	assertContains(t, pwsh, "@('use', 'u', 'unuse', 'un', 'env', 'e', 'set', 'unset', 'path', 'p', 'status', 'st', '--help')")
	assertNotContains(t, pwsh, "'list'")
}

func TestGenSnippetsQuoteShellMetaChars(t *testing.T) {
	t.Run("bash parses and quotes", func(t *testing.T) {
		gen := NewScriptGenerator(Bash)

		assert.Eq(t, "export FOO='it'\\''s'\n", gen.GenSetEnv("FOO", "it's"))
		assert.Eq(t, "export PATH='/opt/my tools/bin':$PATH\n", gen.GenAddPath("/opt/my tools/bin"))
		assert.Eq(t, "export PATH='/a b'\n", gen.GenSetPath([]string{"/a b"}))

		// 生成的片段必须能被 bash 解析
		script := gen.GenSetEnv("FOO", "it's") +
			gen.GenAddPath("/opt/my tools/bin") +
			gen.GenSetPath([]string{"/a b", "/c"})
		cmd := exec.Command("bash", "-n")
		cmd.Stdin = strings.NewReader(script)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("expected generated snippets to parse, err=%v, output=%s, script=%q", err, out, script)
		}
	})

	t.Run("zsh quotes", func(t *testing.T) {
		gen := NewScriptGenerator(Zsh)

		assert.Eq(t, "export FOO='it'\\''s'\n", gen.GenSetEnv("FOO", "it's"))

		// zsh 不做 Git Bash 路径转换, 这里只断言引号包裹与 PATH 前缀
		script := gen.GenAddPath("/opt/my tools/bin")
		assert.True(t, strings.HasPrefix(script, "export PATH='"), "unexpected script: %s", script)
		assert.True(t, strings.HasSuffix(script, "':$PATH\n"), "unexpected script: %s", script)
		assert.Contains(t, script, "my tools")
	})

	t.Run("pwsh escapes single quotes and avoids interpolation", func(t *testing.T) {
		gen := NewScriptGenerator(Pwsh)

		assert.Eq(t, "$Env:FOO='it''s';\n", gen.GenSetEnv("FOO", "it's"))
		assert.Eq(t, "$Env:PATH='C:\\a$b\\bin;' + $Env:PATH\n", gen.GenAddPath(`C:\a$b\bin`))
		assert.Eq(t, "$Env:PATH='C:\\a b';\n", gen.GenSetPath([]string{`C:\a b`}))
	})

	t.Run("cmd escapes lua string", func(t *testing.T) {
		gen := NewScriptGenerator(Cmd)

		assert.Eq(t, "os.setenv('FOO', 'C:\\\\x')\n\n", gen.GenSetEnv("FOO", `C:\x`))
		assert.Eq(t, "os.setenv('PATH', 'C:\\\\a b;%PATH%')\n", gen.GenAddPath(`C:\a b`))
	})
}

func TestBashAddPathUsesGitBashPathSyntaxOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git Bash path conversion is Windows-specific")
	}

	oldHookShell := xenvcom.HookShell()
	xenvcom.SetHookShell("bash")
	t.Cleanup(func() {
		xenvcom.SetHookShell(oldHookShell)
	})

	script := NewScriptGenerator(Bash).GenAddPath(`D:\work\env\devsdk\gosdk\go1.24.6\bin`)

	assertContains(t, script, "export PATH='/d/work/env/devsdk/gosdk/go1.24.6/bin':$PATH")
}

func TestInstallToProfileWritesHookBlock(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	t.Run("bash", func(t *testing.T) {
		profilePath, err := NewScriptGenerator(Bash).InstallToProfile("")
		assert.Require(t, assert.NoErr(t, err))
		assert.Eq(t, filepath.Join(tempHome, ".bashrc"), profilePath)

		content := readTestProfile(t, profilePath)
		assert.Contains(t, content, "# >>> xenv hook >>>")
		assert.Contains(t, content, `eval "$(xenv shell --type bash)"`)
		assert.Contains(t, content, "# <<< xenv hook <<<")

		// 幂等: 重复安装不会产生第二个托管块
		_, err = NewScriptGenerator(Bash).InstallToProfile("")
		assert.Require(t, assert.NoErr(t, err))
		assert.Eq(t, 1, strings.Count(readTestProfile(t, profilePath), "# >>> xenv hook >>>"))
	})

	t.Run("zsh keeps existing content", func(t *testing.T) {
		profilePath := filepath.Join(tempHome, ".zshrc")
		assert.Require(t, assert.NoErr(t, os.WriteFile(profilePath, []byte("export EDITOR=vim\n"), 0o644)))

		_, err := NewScriptGenerator(Zsh).InstallToProfile("")
		assert.Require(t, assert.NoErr(t, err))

		content := readTestProfile(t, profilePath)
		assert.Contains(t, content, "export EDITOR=vim")
		assert.Contains(t, content, `eval "$(xenv shell --type zsh)"`)
	})

	t.Run("pwsh needs the profile path", func(t *testing.T) {
		_, err := NewScriptGenerator(Pwsh).InstallToProfile("")
		assert.ErrMsgContains(t, err, "pwsh profile path")

		profilePath := filepath.Join(tempHome, "profile.ps1")
		got, err := NewScriptGenerator(Pwsh).InstallToProfile(profilePath)
		assert.Require(t, assert.NoErr(t, err))
		assert.Eq(t, profilePath, got)
		assert.Contains(t, readTestProfile(t, profilePath), "Invoke-Expression (& xenv shell --type pwsh)")
	})

	t.Run("cmd is not supported", func(t *testing.T) {
		_, err := NewScriptGenerator(Cmd).InstallToProfile("")
		assert.ErrMsgContains(t, err, "not supported")
	})
}

func readTestProfile(t *testing.T, filePath string) string {
	t.Helper()

	data, err := os.ReadFile(filePath)
	assert.Require(t, assert.NoErr(t, err))
	return string(data)
}

func TestPwshUnsetEnvIgnoresMissingVariables(t *testing.T) {
	script := NewScriptGenerator(Pwsh).GenUnsetEnv("goroot")

	assertContains(t, script, "Remove-Item Env:GOROOT -ErrorAction SilentlyContinue")
}

func TestGeneratedBashHookQuotesConfigValuesWithShellMetaChars(t *testing.T) {
	params := &models.GenInitScriptParams{
		ShellHooksDir: "~/.config/xenv/hooks",
		Paths:         []string{"/opt/Program Files (x86)/NSIS"},
		Envs:          map[string]string{"sdk_home": "/opt/SDKs/Go (stable)"},
		ShellAliases:  map[string]string{"ll": "ls -la --color=auto"},
	}

	script, err := NewScriptGenerator(Bash).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "-n")
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("expected generated bash hook to parse, err=%v, output=%s", err, out)
	}
	assertContains(t, script, `export SDK_HOME='/opt/SDKs/Go (stable)'`)
	assertContains(t, script, `export PATH='/opt/Program Files (x86)/NSIS':$PATH`)
	assertContains(t, script, `alias ll='ls -la --color=auto'`)
}

func TestBashSetPathUsesGitBashPathSyntaxOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git Bash path conversion is Windows-specific")
	}

	oldHookShell := xenvcom.HookShell()
	xenvcom.SetHookShell("bash")
	t.Cleanup(func() {
		xenvcom.SetHookShell(oldHookShell)
	})

	script := NewScriptGenerator(Bash).GenSetPath([]string{
		`D:\work\env\devsdk\gosdk\go1.24.6\bin`,
		`C:\Users\inhere\.xenv\shims`,
		"/usr/bin",
	})

	assertContains(t, script, "export PATH='/d/work/env/devsdk/gosdk/go1.24.6/bin:/c/Users/inhere/.xenv/shims:/usr/bin'")
	assertNotContains(t, script, "D:/")
	assertNotContains(t, script, `C:\`)
}

func TestGeneratedBashHookFormatsGlobalPathForGitBashOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git Bash path conversion is Windows-specific")
	}

	oldHookShell := xenvcom.HookShell()
	xenvcom.SetHookShell("")
	t.Setenv("SHELL", "")
	t.Cleanup(func() {
		xenvcom.SetHookShell(oldHookShell)
	})

	script, err := NewScriptGenerator(Bash).GenHookScripts(&models.GenInitScriptParams{
		ShellHooksDir: "~/.config/xenv/hooks",
		Paths: []string{
			`D:\work\env\devsdk\gosdk\go1.21.13\bin`,
		},
	})
	assert.Require(t, assert.NoErr(t, err))

	assert.Contains(t, script, `export PATH='/d/work/env/devsdk/gosdk/go1.21.13/bin':$PATH`)
	assert.NotContains(t, script, `D:\work`)
	assert.NotContains(t, script, `D:/work`)
}

func TestGeneratedBashHookAvoidsArraySyntaxForHookFiles(t *testing.T) {
	params := &models.GenInitScriptParams{ShellHooksDir: "~/.config/xenv/hooks"}
	script, err := NewScriptGenerator(Bash).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}

	assertNotContains(t, script, "hook_files=(")
	assertContains(t, script, `for file in "${HOME}/.config/xenv/hooks"/*.sh; do`)
}

func TestGeneratedBashHookDoesNotExitShellWhenXenvCommandFails(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	oldBinCommand := xenvcom.BinCommand
	oldBinName := xenvcom.BinName
	xenvcom.SetBinCommand("xenv")
	xenvcom.SetBinName("xenv")
	t.Cleanup(func() {
		xenvcom.BinCommand = oldBinCommand
		xenvcom.BinName = oldBinName
	})

	tempDir := t.TempDir()
	hooksDir := filepath.Join(tempDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeXenv := filepath.Join(tempDir, "xenv")
	if err := os.WriteFile(fakeXenv, []byte(`#!/usr/bin/env bash
if [ "$1" = "shell-init-hook" ]; then
  exit 0
fi
echo "fake xenv failure"
exit 1
`), 0o755); err != nil {
		t.Fatal(err)
	}

	hookScript, err := NewScriptGenerator(Bash).GenHookScripts(&models.GenInitScriptParams{ShellHooksDir: hooksDir})
	if err != nil {
		t.Fatal(err)
	}
	hookFile := filepath.Join(tempDir, "xenv-hook.sh")
	if err := os.WriteFile(hookFile, []byte(hookScript), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "-c", `source "$1"; xenv; echo still-alive`, "bash", hookFile)
	cmd.Env = append(os.Environ(), "PATH="+tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected shell to continue after xenv failure, err=%v, output=%s", err, out)
	}
	assertContains(t, string(out), "still-alive")
}

func TestGeneratedPwshHookPassesLeadingDashArguments(t *testing.T) {
	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh is not available")
	}

	oldBinCommand := xenvcom.BinCommand
	oldBinName := xenvcom.BinName
	xenvcom.SetBinCommand("xenv")
	xenvcom.SetBinName("xenv")
	t.Cleanup(func() {
		xenvcom.BinCommand = oldBinCommand
		xenvcom.BinName = oldBinName
	})

	tempDir := t.TempDir()
	hooksDir := filepath.Join(tempDir, "hooks")
	assert.Require(t, assert.NoErr(t, os.MkdirAll(hooksDir, 0o755)))

	fakeXenv := filepath.Join(tempDir, "xenv")
	fakeContents := "#!/usr/bin/env sh\nif [ \"$1\" = \"shell-init-hook\" ]; then exit 0; fi\necho \"ARGS:$*\"\n"
	if runtime.GOOS == "windows" {
		fakeXenv += ".cmd"
		fakeContents = "@echo off\r\nif \"%1\"==\"shell-init-hook\" exit /b 0\r\necho ARGS:%*\r\n"
	}
	assert.Require(t, assert.NoErr(t, os.WriteFile(fakeXenv, []byte(fakeContents), 0o755)))

	hookScript, err := NewScriptGenerator(Pwsh).GenHookScripts(&models.GenInitScriptParams{ShellHooksDir: hooksDir})
	assert.Require(t, assert.NoErr(t, err))
	hookFile := filepath.Join(tempDir, "xenv-hook.ps1")
	assert.Require(t, assert.NoErr(t, os.WriteFile(hookFile, []byte(hookScript), 0o644)))

	cmd := exec.Command("pwsh", "-NoProfile", "-Command", `. $env:HOOK_SCRIPT; xenv -V`)
	cmd.Env = append(os.Environ(),
		"HOOK_SCRIPT="+hookFile,
		"PATH="+tempDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected pwsh hook to run, err=%v, output=%s", err, out)
	}
	if !strings.Contains(string(out), "ARGS:-V") {
		t.Fatalf("expected pwsh hook to pass -V to xenv binary, output=%s", out)
	}
}

func TestGeneratedPwshHookKeepsShellAliasesAfterSetup(t *testing.T) {
	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh is not available")
	}

	oldBinCommand := xenvcom.BinCommand
	oldBinName := xenvcom.BinName
	xenvcom.SetBinCommand("xenv")
	xenvcom.SetBinName("xenv")
	t.Cleanup(func() {
		xenvcom.BinCommand = oldBinCommand
		xenvcom.BinName = oldBinName
	})

	tempDir := t.TempDir()
	hooksDir := filepath.Join(tempDir, "hooks")
	assert.Require(t, assert.NoErr(t, os.MkdirAll(hooksDir, 0o755)))

	fakeXenv := filepath.Join(tempDir, "xenv")
	fakeContents := "#!/usr/bin/env sh\nif [ \"$1\" = \"shell-init-hook\" ]; then exit 0; fi\n"
	if runtime.GOOS == "windows" {
		fakeXenv += ".cmd"
		fakeContents = "@echo off\r\nif \"%1\"==\"shell-init-hook\" exit /b 0\r\n"
	}
	assert.Require(t, assert.NoErr(t, os.WriteFile(fakeXenv, []byte(fakeContents), 0o755)))

	hookScript, err := NewScriptGenerator(Pwsh).GenHookScripts(&models.GenInitScriptParams{
		ShellHooksDir: hooksDir,
		ShellAliases: map[string]string{
			"adb": "Get-ChildItem",
			"llx": "Get-ChildItem -Force",
		},
	})
	assert.Require(t, assert.NoErr(t, err))
	hookFile := filepath.Join(tempDir, "xenv-hook.ps1")
	assert.Require(t, assert.NoErr(t, os.WriteFile(hookFile, []byte(hookScript), 0o644)))

	cmd := exec.Command("pwsh", "-NoProfile", "-Command", `. $env:HOOK_SCRIPT; Get-Command adb,llx | Select-Object -ExpandProperty Name`)
	cmd.Env = append(os.Environ(),
		"HOOK_SCRIPT="+hookFile,
		"PATH="+tempDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected pwsh hook aliases to remain available, err=%v, output=%s", err, out)
	}
	assertContains(t, string(out), "adb")
	assertContains(t, string(out), "llx")
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected generated script to contain %q", substr)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Fatalf("expected generated script not to contain %q", substr)
	}
}
