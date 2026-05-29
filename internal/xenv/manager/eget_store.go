package manager

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/inhere/xenv/internal/xenv/models"
)

type EgetStoreSource struct {
	Path string
}

type egetInstalledStore struct {
	Schema    int                                  `json:"schema"`
	Installed map[string]egetInstalledStoreToolSet `json:"installed"`
}

type egetInstalledStoreToolSet struct {
	Versions map[string]egetInstalledStoreVersion `json:"versions"`
}

type egetInstalledStoreVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

func (s EgetStoreSource) ListSDKVersions(name string) ([]models.InstalledSDK, error) {
	if s.Path == "" {
		return nil, nil
	}

	store, err := s.load()
	if err != nil {
		return nil, err
	}

	item, ok := store.Installed[name]
	if !ok || len(item.Versions) == 0 {
		return nil, nil
	}

	items := make([]models.InstalledSDK, 0, len(item.Versions))
	for version, meta := range item.Versions {
		sdkName := meta.Name
		if sdkName == "" {
			sdkName = name
		}

		sdkVersion := meta.Version
		if sdkVersion == "" {
			sdkVersion = version
		}

		items = append(items, models.InstalledSDK{
			ID:         sdkName + ":" + sdkVersion,
			Name:       sdkName,
			Version:    sdkVersion,
			InstallDir: meta.Path,
			Source:     "eget",
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Version > items[j].Version
	})
	return items, nil
}

func (s EgetStoreSource) load() (*egetInstalledStore, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, err
	}

	store := &egetInstalledStore{}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, err
	}
	return store, nil
}
