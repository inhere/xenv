package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/jsonutil"
	"github.com/inhere/xenv/internal/xenv/manager"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

func TestSetupDirenvDetectsGoModWithoutCreatingXenvToml(t *testing.T) {
	tempHome, projectDir, svc, state := newDirenvTestService(t, "test-session", func(projectDir string) {
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/test\n\ngo 1.24\n"), 0644); err != nil {
			t.Fatal(err)
		}
	})

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	if script == "" {
		t.Fatal("expected setup direnv to generate activation script")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".xenv.toml")); !os.IsNotExist(err) {
		t.Fatalf("expected .xenv.toml not to be created, stat err=%v", err)
	}
	sessionID := xenvcom.SessionIDForDir(projectDir)
	if _, err := os.Stat(filepath.Join(tempHome, ".xenv", "session", sessionID+".json")); err != nil {
		t.Fatalf("expected session state to be saved: %v", err)
	}
	if state.Nearest() != nil {
		t.Fatal("expected auto-detected project files not to create direnv state")
	}
}

func TestSetupDirenvUsesExistingXenvTomlAsDirenvState(t *testing.T) {
	_, _, svc, state := newDirenvTestService(t, "test-existing-direnv", func(projectDir string) {
		xenvToml := filepath.Join(projectDir, ".xenv.toml")
		if err := os.WriteFile(xenvToml, []byte("paths = []\n\n[sdks]\n  go = \"1.24\"\n\n[envs]\n\n[tools]\n"), 0644); err != nil {
			t.Fatal(err)
		}
	})

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	if script == "" {
		t.Fatal("expected existing .xenv.toml to generate activation script")
	}
	if state.Nearest() == nil {
		t.Fatal("expected existing .xenv.toml to be loaded as direnv state")
	}
}

func TestSetupDirenvDetectsProjectRootGoModFromSubdirectory(t *testing.T) {
	tempHome, projectDir, svc, state := newDirenvTestService(t, "test-subdir", func(projectDir string) {
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/test\n\ngo 1.24\n"), 0644); err != nil {
			t.Fatal(err)
		}
		subDir := filepath.Join(projectDir, "pkg", "service")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(subDir); err != nil {
			t.Fatal(err)
		}
	})

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	if script == "" {
		t.Fatal("expected setup direnv from subdirectory to detect project root go.mod")
	}
	if _, err := os.Stat(filepath.Join(projectDir, "pkg", "service", ".xenv.toml")); !os.IsNotExist(err) {
		t.Fatalf("expected subdirectory .xenv.toml not to be created, stat err=%v", err)
	}
	sessionID := xenvcom.SessionIDForDir(projectDir)
	if _, err := os.Stat(filepath.Join(tempHome, ".xenv", "session", sessionID+".json")); err != nil {
		t.Fatalf("expected project-root session state to be saved: %v", err)
	}
	if state.Nearest() != nil {
		t.Fatal("expected auto-detected project root files not to create direnv state")
	}
}

func newDirenvTestService(t *testing.T, sessionID string, setupProject func(projectDir string)) (tempHome, projectDir string, svc *ToolService, state *manager.StateManager) {
	t.Helper()

	tempHome = t.TempDir()
	projectDir = t.TempDir()
	installDir := filepath.Join(tempHome, "tools", "go", "1.24")

	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)
	t.Setenv("XENV_HOOK_SHELL", "pwsh")
	xenvcom.SetHookShell("pwsh")
	xenvcom.SetSessionID("")
	t.Cleanup(func() {
		xenvcom.SetHookShell("")
		xenvcom.SetSessionID("")
	})

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if setupProject != nil {
		setupProject(projectDir)
	}

	localToolsFile := filepath.Join(tempHome, ".config", "xenv", "sdks.local.json")
	if err := os.MkdirAll(filepath.Dir(localToolsFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := jsonutil.WritePretty(localToolsFile, &models.SDKLocalIndex{
		Schema: 1,
		SDKs: []models.InstalledSDK{{
			ID:         "go:1.24",
			Name:       "go",
			Version:    "1.24",
			InstallDir: installDir,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	cfg := &models.Configuration{
		SDKs: []models.ToolChain{{
			Name: "go",
		}},
	}
	state = manager.NewStateManager()
	if err := state.Init(); err != nil {
		t.Fatal(err)
	}
	toolMgr := manager.NewSDKManager(localToolsFile)
	if err := toolMgr.Init(cfg); err != nil {
		t.Fatal(err)
	}
	return tempHome, projectDir, NewToolService(cfg, state, toolMgr), state
}
