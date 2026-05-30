package models

import "testing"

func TestSDKLocalIndexFindByIDReturnsStoredSDK(t *testing.T) {
	idx := NewSDKLocalIndex()
	idx.SDKs = []InstalledSDK{{
		ID:         "go:1.22.0",
		Name:       "go",
		Version:    "1.22.0",
		InstallDir: "/sdk/go1.22.0",
		Source:     "xenv",
	}}

	got := idx.FindByID("go:1.22.0")
	if got == nil {
		t.Fatal("expected SDK")
	}
	got.Config = &ToolChain{Name: "go", BinDir: "bin"}

	if idx.SDKs[0].Config == nil {
		t.Fatal("FindByID must return the stored SDK, not a range copy")
	}
	if got.Index != 0 {
		t.Fatalf("Index = %d, want 0", got.Index)
	}
}
