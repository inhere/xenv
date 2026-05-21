package cli

import "testing"

func TestNewAppRegistersTopLevelCommands(t *testing.T) {
	app := NewApp()

	if app.Name != "xenv" {
		t.Fatalf("expected app name xenv, got %q", app.Name)
	}

	for _, name := range []string{
		"tools",
		"use",
		"unuse",
		"env",
		"path",
		"config",
		"list",
		"init",
		"shell",
		"shell-init-hook",
		"shell-direnv",
	} {
		if !app.HasCommand(name) {
			t.Fatalf("expected app to register command %q", name)
		}
	}

	if app.ResolveAlias("init-direnv") != "shell-direnv" {
		t.Fatalf("expected init-direnv alias to resolve to shell-direnv")
	}
}
