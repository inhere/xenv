package xenvcom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionIDUsesProjectRootWhenEnvIsUnset(t *testing.T) {
	oldSessionID := sessionID
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	sessionID = ""
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	t.Cleanup(func() {
		sessionID = oldSessionID
	})

	projectDir := t.TempDir()
	subDir := filepath.Join(projectDir, "pkg", "service")
	otherDir := t.TempDir()
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	rootID := SessionID()

	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	subDirID := SessionID()

	if rootID != subDirID {
		t.Fatalf("expected project subdirectory to share root session ID, root=%q subdir=%q", rootID, subDirID)
	}
	if !strings.HasPrefix(rootID, filepath.Base(projectDir)+"_") {
		t.Fatalf("expected session ID %q to include project root name %q", rootID, filepath.Base(projectDir))
	}

	if err := os.Chdir(otherDir); err != nil {
		t.Fatal(err)
	}
	otherID := SessionID()
	if rootID == otherID {
		t.Fatalf("expected different project roots to use different session IDs, got %q", rootID)
	}
}

func TestSessionIDUsesExplicitEnvValueWhenSet(t *testing.T) {
	oldSessionID := sessionID
	sessionID = "explicit-session"
	t.Cleanup(func() {
		sessionID = oldSessionID
	})

	if got := SessionID(); got != "explicit-session" {
		t.Fatalf("expected explicit session ID, got %q", got)
	}
}
