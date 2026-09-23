package models

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

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

func TestSDKLocalIndexListByNameSortsByVersionDesc(t *testing.T) {
	idx := NewSDKLocalIndex()
	idx.SDKs = []InstalledSDK{
		{ID: "go:1.9.0", Name: "go", Version: "1.9.0", InstallDir: "D:/sdk/go1.9.0"},
		{ID: "go:1.26.10", Name: "go", Version: "1.26.10", InstallDir: "D:/sdk/go1.26.10"},
		{ID: "go:1.10.0", Name: "go", Version: "1.10.0", InstallDir: "D:/sdk/go1.10.0"},
		{ID: "go:1.26.9", Name: "go", Version: "1.26.9", InstallDir: "D:/sdk/go1.26.9"},
		{ID: "go:1.26.0-rc1", Name: "go", Version: "1.26.0-rc1", InstallDir: "D:/sdk/go1.26.0-rc1"},
		{ID: "node:22.1.0", Name: "node", Version: "22.1.0", InstallDir: "D:/sdk/node22.1.0"},
	}

	items := idx.ListByName("go")

	versions := make([]string, 0, len(items))
	indexes := make([]int, 0, len(items))
	for _, item := range items {
		versions = append(versions, item.Version)
		indexes = append(indexes, item.Index)
	}

	// 新 -> 旧，预发布版本排在正式版之后
	assert.Eq(t, []string{"1.26.10", "1.26.9", "1.26.0-rc1", "1.10.0", "1.9.0"}, versions)
	// Index 仍指向原始 SDKs 中的位置
	assert.Eq(t, []int{1, 3, 4, 2, 0}, indexes)
}
