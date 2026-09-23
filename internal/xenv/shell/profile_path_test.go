package shell

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestProfilePath(t *testing.T) {
	assert.Eq(t, "~/.bashrc", Bash.ProfilePath())
	assert.Eq(t, "~/.zshrc", Zsh.ProfilePath())
	assert.Eq(t, "~/.pwsh/profile.ps1", Pwsh.ProfilePath())
	assert.Eq(t, "~/AppData/Local/clink/profile.lua", Cmd.ProfilePath())
}
