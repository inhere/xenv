package shell

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestTypeFromStringAcceptsShellExecutablePaths(t *testing.T) {
	tests := map[string]ShType{
		"bash":                                   Bash,
		"C:/Program Files/Git/usr/bin/bash":      Bash,
		`C:\Program Files\Git\usr\bin\bash.exe`:  Bash,
		"/usr/bin/zsh":                           Zsh,
		"C:/Program Files/PowerShell/7/pwsh.exe": Pwsh,
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := TypeFromString(input)
			assert.Require(t, assert.NoErr(t, err))
			assert.Eq(t, want, got)
		})
	}
}
