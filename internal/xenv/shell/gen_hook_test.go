package shell

import (
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
