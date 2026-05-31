package cli

import "testing"

func TestNewAppRegistersTopLevelCommands(t *testing.T) {
	app := NewApp()

	if app.Name != "xenv" {
		t.Fatalf("expected app name xenv, got %q", app.Name)
	}

	for _, name := range []string{
		"sdk",
		"check",
		"use",
		"unuse",
		"env",
		"path",
		"config",
		"list",
		"shell",
		"shell-init-hook",
		"shell-direnv",
	} {
		if !app.HasCommand(name) {
			t.Fatalf("expected app to register command %q", name)
		}
	}

	if app.HasCommand("tools") {
		t.Fatalf("tools command must not be registered")
	}
	if app.HasCommand("init") {
		t.Fatalf("init command must be registered under config, not at top level")
	}
	configCmd := app.GetCommand("config")
	if configCmd == nil {
		t.Fatal("expected config command to be registered")
	}
	if configCmd.Commands()["init"] == nil {
		t.Fatalf("expected config command to register init subcommand")
	}

	if app.ResolveAlias("init-direnv") != "shell-direnv" {
		t.Fatalf("expected init-direnv alias to resolve to shell-direnv")
	}
}
