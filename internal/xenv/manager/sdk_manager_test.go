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
