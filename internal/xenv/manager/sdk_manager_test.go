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
