package models

import (
	"sort"
	"time"

	"github.com/inhere/xenv/internal/util"
)

type SDKLocalIndex struct {
	Schema    int            `json:"schema"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	SDKs      []InstalledSDK `json:"sdks"`
}

type InstalledSDK struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	InstallDir string    `json:"install_dir"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Index      int        `json:"-"`
	Config     *ToolChain `json:"-"`
}

func NewSDKLocalIndex() *SDKLocalIndex {
	return &SDKLocalIndex{Schema: 1}
}

func (idx *SDKLocalIndex) ListByName(name string) []InstalledSDK {
	var items []InstalledSDK
	for i, sdk := range idx.SDKs {
		if sdk.Name == name {
			sdk.Index = i
			items = append(items, sdk)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Version > items[j].Version
	})
	return items
}

func (idx *SDKLocalIndex) FindByID(id string) *InstalledSDK {
	for i, sdk := range idx.SDKs {
		if sdk.ID == id {
			sdk.Index = i
			return &sdk
		}
	}
	return nil
}

func (s *InstalledSDK) BinDirPath() string {
	if s.Config == nil {
		return util.NormalizePath(s.InstallDir)
	}
	return s.Config.FullBinPath(s.InstallDir)
}

func (s *InstalledSDK) RenderActiveEnv() map[string]string {
	if s.Config == nil || len(s.Config.ActiveEnv) == 0 {
		return nil
	}
	return s.Config.RenderActiveEnv(map[string]string{
		"name":        s.Name,
		"version":     s.Version,
		"install_dir": util.NormalizePath(s.InstallDir),
	})
}
