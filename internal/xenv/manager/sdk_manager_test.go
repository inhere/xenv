package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inhere/xenv/internal/xenv/models"
)

func TestSDKManagerIndexLocalSDKsWritesSDKOnlyIndex(t *testing.T) {
	root := t.TempDir()
	go122 := filepath.Join(root, "go1.22.0")
	go123 := filepath.Join(root, "go1.23.1")
	if err := os.MkdirAll(filepath.Join(go122, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(go123, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	indexFile := filepath.Join(t.TempDir(), "sdks.local.json")
	mgr := NewSDKManager(indexFile)
	err := mgr.Init(&models.Configuration{
		SDKs: []models.ToolChain{{
			Name:       "go",
			InstallDir: filepath.Join(root, "go{version}"),
			BinDir:     "bin",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.IndexLocalSDKs(); err != nil {
		t.Fatal(err)
	}

	idx, err := mgr.LoadLocalIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.SDKs) != 2 {
		t.Fatalf("indexed SDKs = %d, want 2", len(idx.SDKs))
	}

	data, err := os.ReadFile(indexFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"tools"`) {
		t.Fatal("unexpected tools field")
	}
}

func TestSDKManagerRequiresInjectedIndexFile(t *testing.T) {
	mgr := NewSDKManager("")
	if _, err := mgr.LoadLocalIndex(); err == nil {
		t.Fatal("expected missing index file path error")
	}
}

func TestSDKManagerCanRetryLoadAfterFailure(t *testing.T) {
	indexFile := filepath.Join(t.TempDir(), "sdks.local.json")
	mgr := NewSDKManager(indexFile)

	if err := os.WriteFile(indexFile, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.LoadLocalIndex(); err == nil {
		t.Fatal("expected invalid json error")
	}

	data := []byte(`{"schema":1,"sdks":[{"id":"go:1.22.0","name":"go","version":"1.22.0","install_dir":"/sdk/go1.22.0","source":"xenv"}]}`)
	if err := os.WriteFile(indexFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := mgr.LoadLocalIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.SDKs) != 1 {
		t.Fatalf("loaded SDKs = %d, want 1", len(idx.SDKs))
	}
}

func TestSDKManagerListMergedSDKVersionsPrefersEget(t *testing.T) {
	indexFile := filepath.Join(t.TempDir(), "sdks.local.json")
	storeFile := filepath.Join(t.TempDir(), "sdk.installed.json")

	localData := []byte(`{"schema":1,"sdks":[{"id":"go:1.22.0","name":"go","version":"1.22.0","install_dir":"D:/xenv/go1.22.0","source":"xenv"},{"id":"go:1.23.0","name":"go","version":"1.23.0","install_dir":"D:/xenv/go1.23.0","source":"xenv"}]}`)
	if err := os.WriteFile(indexFile, localData, 0o644); err != nil {
		t.Fatal(err)
	}

	storeData := []byte(`{
	  "schema": 1,
	  "installed": {
	    "go": {
	      "versions": {
	        "1.22.0": {
	          "name": "go",
	          "version": "1.22.0",
	          "path": "D:/eget/go1.22.0"
	        }
	      }
	    }
	  }
	}`)
	if err := os.WriteFile(storeFile, storeData, 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewSDKManager(indexFile)
	if err := mgr.Init(&models.Configuration{
		EgetEnable:    true,
		EgetStoreFile: storeFile,
	}); err != nil {
		t.Fatal(err)
	}
	mgr.SetEgetSource(EgetStoreSource{Path: storeFile})

	items := mgr.ListMergedSDKVersions("go")
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Version != "1.23.0" {
		t.Fatalf("first version = %q, want 1.23.0", items[0].Version)
	}
	if items[1].Version != "1.22.0" {
		t.Fatalf("second version = %q, want 1.22.0", items[1].Version)
	}
	if items[1].Source != "eget" {
		t.Fatalf("source = %q, want eget", items[1].Source)
	}
	if items[1].InstallDir != "D:/eget/go1.22.0" {
		t.Fatalf("install dir = %q", items[1].InstallDir)
	}
}

func TestSDKManagerListMergedSDKVersionsFallsBackWhenEgetStoreMissing(t *testing.T) {
	indexFile := filepath.Join(t.TempDir(), "sdks.local.json")
	localData := []byte(`{"schema":1,"sdks":[{"id":"go:1.23.0","name":"go","version":"1.23.0","install_dir":"D:/xenv/go1.23.0","source":"xenv"}]}`)
	if err := os.WriteFile(indexFile, localData, 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewSDKManager(indexFile)
	if err := mgr.Init(&models.Configuration{
		EgetEnable:    true,
		EgetStoreFile: filepath.Join(t.TempDir(), "missing.json"),
	}); err != nil {
		t.Fatal(err)
	}

	items := mgr.ListMergedSDKVersions("go")
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Source != "xenv" {
		t.Fatalf("source = %q, want xenv", items[0].Source)
	}
}
