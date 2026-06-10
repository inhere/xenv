package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
		assertContains(t, script, `$script:XenvBinCommand = (Get-Command xenv -CommandType Application -ErrorAction Stop).Source`)
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
	assertContains(t, bash, `complete -W "use u unuse un env e set unset path p list help" xenv`)

	zsh, err := NewScriptGenerator(Zsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, zsh, "use|u|unuse|un|env|e|path|p)")
	assertContains(t, zsh, `compctl -k "use u unuse un env e set unset path p list help" xenv`)

	pwsh, err := NewScriptGenerator(Pwsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, pwsh, "{ $_ -in @('use', 'u', 'unuse', 'un', 'env', 'e', 'path', 'p') }")
	assertContains(t, pwsh, "@('use', 'u', 'unuse', 'un', 'env', 'e', 'set', 'unset', 'path', 'p', 'list', '--help')")
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
