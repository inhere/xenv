package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	if _, err := os.Stat(filepath.Join(tempHome, ".config", "xenv", "session", sessionID+".json")); err != nil {
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

func TestSetupDirenvDoesNotRewriteExistingXenvTomlSDKVersion(t *testing.T) {
	_, projectDir, svc, _ := newDirenvTestService(t, "test-existing-direnv-preserve-sdk-spec", func(projectDir string) {
		xenvToml := filepath.Join(projectDir, ".xenv.toml")
		if err := os.WriteFile(xenvToml, []byte("paths = []\n\n[sdks]\n  go = \"1.24\" # keep partial version\n\n[envs]\n\n[tools]\n"), 0o644); err != nil {
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

	data, err := os.ReadFile(filepath.Join(projectDir, ".xenv.toml"))
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	if !strings.Contains(contents, `go = "1.24" # keep partial version`) {
		t.Fatalf("expected SetupDirenv to preserve existing SDK spec, got:\n%s", contents)
	}
	if strings.Contains(contents, `go = "1.24.0"`) {
		t.Fatalf("expected SetupDirenv not to rewrite SDK spec to real version, got:\n%s", contents)
	}
}

func TestSetupDirenvGeneratesEnvAndPathWithoutSDK(t *testing.T) {
	_, _, svc, _ := newDirenvTestService(t, "test-existing-env-path-no-sdk", func(projectDir string) {
		xenvToml := filepath.Join(projectDir, ".xenv.toml")
		data := "paths = [\"./bin\"]\n\n[envs]\n  foo = \"bar\"\n\n[tools]\n"
		if err := os.WriteFile(xenvToml, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	})

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	if !containsNormalized(script, "./bin") {
		t.Fatalf("expected setup direnv script to add project path, got %q", script)
	}
	if !strings.Contains(script, "$Env:FOO='bar';") {
		t.Fatalf("expected setup direnv script to set project env, got %q", script)
	}
}

func TestSetupDirenvAppendsProjectScriptWithoutSDK(t *testing.T) {
	_, projectDir, svc, _ := newDirenvTestService(t, "test-existing-project-script-no-sdk", func(projectDir string) {
		xenvToml := filepath.Join(projectDir, ".xenv.toml")
		if err := os.WriteFile(xenvToml, []byte("paths = []\n\n[envs]\n\n[tools]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projectDir, ".xenv.ps1"), []byte("# project hook\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	svc.config.SourceProjectScripts = true

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	want := `. "` + filepath.ToSlash(projectDir) + `/.xenv.ps1"`
	if !containsNormalized(script, want) {
		t.Fatalf("expected setup direnv script to source project pwsh script without SDK, want %q, got %q", want, script)
	}
}

func TestSetupDirenvAppendsProjectScriptForPwsh(t *testing.T) {
	_, projectDir, svc, _ := newDirenvTestService(t, "test-existing-project-script-pwsh", func(projectDir string) {
		xenvToml := filepath.Join(projectDir, ".xenv.toml")
		if err := os.WriteFile(xenvToml, []byte("paths = []\n\n[sdks]\n  go = \"1.24\"\n\n[envs]\n\n[tools]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projectDir, ".xenv.ps1"), []byte("# project hook\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	svc.config.SourceProjectScripts = true

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	want := `. "` + filepath.ToSlash(projectDir) + `/.xenv.ps1"`
	if !containsNormalized(script, want) {
		t.Fatalf("expected setup direnv script to source project pwsh script, want %q, got %q", want, script)
	}
}

func TestSetupDirenvAppendsProjectScriptForBash(t *testing.T) {
	_, projectDir, svc, _ := newDirenvTestService(t, "test-existing-project-script-bash", func(projectDir string) {
		xenvToml := filepath.Join(projectDir, ".xenv.toml")
		if err := os.WriteFile(xenvToml, []byte("paths = []\n\n[sdks]\n  go = \"1.24\"\n\n[envs]\n\n[tools]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projectDir, ".xenv.sh"), []byte("# project hook\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("XENV_HOOK_SHELL", "bash")
		xenvcom.SetHookShell("bash")
	})
	svc.config.SourceProjectScripts = true

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	want := `source "` + filepath.ToSlash(projectDir) + `/.xenv.sh"`
	if !containsNormalized(script, want) {
		t.Fatalf("expected setup direnv script to source project bash script, want %q, got %q", want, script)
	}
}

func TestSetupDirenvSkipsProjectScriptWithoutDirenvState(t *testing.T) {
	_, _, svc, _ := newDirenvTestService(t, "test-no-direnv-script", func(projectDir string) {
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/test\n\ngo 1.24\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projectDir, ".xenv.ps1"), []byte("# project hook\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	svc.config.SourceProjectScripts = true

	script, err := svc.SetupDirenv()
	if err != nil {
		t.Fatal(err)
	}
	if containsNormalized(script, ".xenv.ps1") || containsNormalized(script, ".xenv.sh") {
		t.Fatalf("expected setup direnv script not to source project script without direnv state, got %q", script)
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
	if _, err := os.Stat(filepath.Join(tempHome, ".config", "xenv", "session", sessionID+".json")); err != nil {
		t.Fatalf("expected project-root session state to be saved: %v", err)
	}
	if state.Nearest() != nil {
		t.Fatal("expected auto-detected project root files not to create direnv state")
	}
}

func TestActivateSDKsStoresMatchedVersion(t *testing.T) {
	tempHome, projectDir, svc, state := newDirenvTestService(t, "test-real-version", nil)

	if _, err := svc.ActivateSDKs([]string{"go:1.24"}, models.OpFlagSession); err != nil {
		t.Fatal(err)
	}

	if got := state.Merged().SDKs["go"]; got != "1.24.0" {
		t.Fatalf("merged go version = %q, want %q", got, "1.24.0")
	}

	sessionID := xenvcom.SessionIDForDir(projectDir)
	stateFile := filepath.Join(tempHome, ".config", "xenv", "session", sessionID+".json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	var saved models.ActivityState
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if got := saved.SDKs["go"]; got != "1.24.0" {
		t.Fatalf("saved go version = %q, want %q", got, "1.24.0")
	}
}

func TestWhereSDKUsesXenvIndexWhenEgetHasSameVersion(t *testing.T) {
	_, _, svc, _ := newDirenvTestService(t, "test-where-xenv-source", nil)
	svc.config.EgetEnable = true
	svc.sdks.SetEgetSource(manager.EgetStoreSource{
		Path: writeTestEgetStore(t, "go", "1.24.0", "D:/eget/go1.24.0"),
	})

	got, err := svc.WhereSDK("go:1.24.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.ToSlash(got) == "D:/eget/go1.24.0" {
		t.Fatalf("WhereSDK should use xenv local index for activation paths, got eget path %q", got)
	}
	if filepath.Base(got) != "1.24.0" {
		t.Fatalf("WhereSDK() = %q, want local 1.24.0 path", got)
	}
}

func TestWhereSDKUsesEgetWhenOnlyEgetHasVersion(t *testing.T) {
	_, _, svc, _ := newDirenvTestService(t, "test-where-eget-source", nil)
	svc.config.EgetEnable = true
	svc.sdks.SetEgetSource(manager.EgetStoreSource{
		Path: writeTestEgetStore(t, "go", "1.25.0", "D:/eget/go1.25.0"),
	})

	got, err := svc.WhereSDK("go:1.25.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.ToSlash(got) != "D:/eget/go1.25.0" {
		t.Fatalf("WhereSDK() = %q, want eget path", got)
	}
}

func TestDeactivateSDKsUsesEgetSourceWhenOnlyEgetHasVersion(t *testing.T) {
	_, _, svc, state := newDirenvTestService(t, "test-unuse-eget-source", nil)
	svc.config.EgetEnable = true
	svc.config.SDKs = append(svc.config.SDKs, models.ToolChain{Name: "node", BinDir: "bin"})
	svc.sdks.SetEgetSource(manager.EgetStoreSource{
		Path: writeTestEgetStore(t, "node", "22.22.3", "D:/eget/node22.22.3"),
	})

	useScript, err := svc.ActivateSDKs([]string{"node:22"}, models.OpFlagSession)
	if err != nil {
		t.Fatal(err)
	}
	if !containsNormalized(useScript, "D:/eget/node22.22.3/bin") {
		t.Fatalf("expected use script to add eget node bin path, got %q", useScript)
	}
	if got := state.Merged().SDKs["node"]; got != "22.22.3" {
		t.Fatalf("merged node version after use = %q, want %q", got, "22.22.3")
	}

	t.Setenv("PATH", filepath.FromSlash("D:/eget/node22.22.3/bin")+string(os.PathListSeparator)+filepath.FromSlash("D:/tools/keep"))
	unuseScript, err := svc.DeactivateSDKs([]string{"node:22"}, models.OpFlagSession)
	if err != nil {
		t.Fatal(err)
	}
	if containsNormalized(unuseScript, "D:/eget/node22.22.3/bin") {
		t.Fatalf("expected unuse script to remove eget node bin path, got %q", unuseScript)
	}
	if !containsNormalized(unuseScript, "D:/tools/keep") {
		t.Fatalf("expected unuse script to preserve other PATH entries, got %q", unuseScript)
	}
	if got := state.Merged().SDKs["node"]; got != "" {
		t.Fatalf("merged node version after unuse = %q, want empty", got)
	}
}

func writeTestEgetStore(t *testing.T, name, version, installDir string) string {
	t.Helper()

	storeFile := filepath.Join(t.TempDir(), "sdk.installed.json")
	data := []byte(`{
	  "schema": 1,
	  "installed": {
	    "` + name + `": {
	      "versions": {
	        "` + version + `": {
	          "name": "` + name + `",
	          "version": "` + version + `",
	          "path": "` + installDir + `"
	        }
	      }
	    }
	  }
	}`)
	if err := os.WriteFile(storeFile, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return storeFile
}

func newDirenvTestService(t *testing.T, sessionID string, setupProject func(projectDir string)) (tempHome, projectDir string, svc *SDKService, state *manager.StateManager) {
	t.Helper()

	tempHome = t.TempDir()
	projectDir = t.TempDir()
	installDir := filepath.Join(tempHome, "tools", "go", "1.24.0")

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
			ID:         "go:1.24.0",
			Name:       "go",
			Version:    "1.24.0",
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
	return tempHome, projectDir, NewSDKService(cfg, state, toolMgr), state
}

func containsNormalized(s, substr string) bool {
	return strings.Contains(filepath.ToSlash(s), filepath.ToSlash(substr))
}
