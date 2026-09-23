package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestEgetStoreSourceListsSDKs(t *testing.T) {
	store := filepath.Join(t.TempDir(), "sdk.installed.json")
	data := []byte(`{
	  "schema": 1,
	  "installed": {
	    "go": {
	      "versions": {
	        "1.22.0": {
	          "name": "go",
	          "version": "1.22.0",
	          "path": "D:/sdk/go1.22.0"
	        }
	      }
	    }
	  }
	}`)
	if err := os.WriteFile(store, data, 0o644); err != nil {
		t.Fatal(err)
	}

	src := EgetStoreSource{Path: store}
	items, err := src.ListSDKVersions("go")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Source != "eget" {
		t.Fatalf("source = %q, want eget", items[0].Source)
	}
	if items[0].InstallDir != "D:/sdk/go1.22.0" {
		t.Fatalf("install dir = %q", items[0].InstallDir)
	}
}

func TestEgetStoreSourceEmptyPathReturnsEmpty(t *testing.T) {
	src := EgetStoreSource{}

	items, err := src.ListSDKVersions("go")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestDefaultEgetStoreFileUsesHomeConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	want := filepath.Join(home, ".config", "eget", "sdk.installed.json")
	if got := DefaultEgetStoreFile(); got != want {
		t.Fatalf("DefaultEgetStoreFile() = %q, want %q", got, want)
	}
}

func TestEgetStoreSourceListSDKVersionsSortsByVersionDesc(t *testing.T) {
	store := filepath.Join(t.TempDir(), "sdk.installed.json")
	data := []byte(`{
	  "schema": 1,
	  "installed": {
	    "go": {
	      "versions": {
	        "1.9.0": {"name": "go", "version": "1.9.0", "path": "D:/eget/go1.9.0"},
	        "1.26.10": {"name": "go", "version": "1.26.10", "path": "D:/eget/go1.26.10"},
	        "1.10.0": {"name": "go", "version": "1.10.0", "path": "D:/eget/go1.10.0"},
	        "1.26.9": {"name": "go", "version": "1.26.9", "path": "D:/eget/go1.26.9"}
	      }
	    }
	  }
	}`)
	if err := os.WriteFile(store, data, 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := EgetStoreSource{Path: store}.ListSDKVersions("go")
	assert.Require(t, assert.NoErr(t, err))

	versions := make([]string, 0, len(items))
	for _, item := range items {
		versions = append(versions, item.Version)
	}

	// 新 -> 旧，数字段按数值比较而非字典序
	assert.Eq(t, []string{"1.26.10", "1.26.9", "1.10.0", "1.9.0"}, versions)
}
