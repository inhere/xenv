package manager

import (
	"os"
	"path/filepath"
	"testing"
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

func TestEgetStoreSourceMissingFileReturnsEmpty(t *testing.T) {
	src := EgetStoreSource{}

	items, err := src.ListSDKVersions("go")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}
