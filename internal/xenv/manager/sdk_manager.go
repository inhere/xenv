package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/jsonutil"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
	"github.com/inhere/xenv/internal/xenv/xenvutil"
)

type SDKManager struct {
	init bool

	config *models.Configuration

	localLoad bool
	localFile string
	localSDKs *models.SDKLocalIndex
	egetSrc   EgetStoreSource

	groupSDKs map[string][]models.InstalledSDK
}

func NewSDKManager(indexFile string) *SDKManager {
	return &SDKManager{
		localFile: indexFile,
		localSDKs: models.NewSDKLocalIndex(),
		groupSDKs: make(map[string][]models.InstalledSDK),
	}
}

func (m *SDKManager) Init(config *models.Configuration) error {
	if m.init {
		return nil
	}
	m.init = true
	m.config = config
	if m.egetSrc.Path == "" && config != nil {
		storeFile := config.EgetStoreFile
		if config.EgetEnable && storeFile == "" {
			storeFile = DefaultEgetStoreFile()
		}
		m.egetSrc = EgetStoreSource{Path: storeFile}
	}
	return nil
}

func (m *SDKManager) SetEgetSource(source EgetStoreSource) {
	m.egetSrc = source
}

func (m *SDKManager) InitLoad() error {
	return m.ensureLocalLoad(false)
}

func (m *SDKManager) ensureLocalLoad(must bool) error {
	if m.localLoad {
		return nil
	}

	err := m.LoadLocalIndexIntoCache()
	if err != nil && must {
		panic(err)
	}
	if err == nil {
		m.localLoad = true
	}
	return err
}

func (m *SDKManager) LoadLocalIndexIntoCache() error {
	if m.localFile == "" {
		return fmt.Errorf("sdk local index file is required")
	}

	fileExist := fsutil.IsFile(m.localFile)
	xenvcom.Debugf("Load local index file: %s(exist=%v)\n", m.localFile, fileExist)

	if fileExist {
		if err := jsonutil.DecodeFile(m.localFile, m.localSDKs); err != nil {
			return err
		}
	}
	return nil
}

func (m *SDKManager) LoadLocalIndex() (*models.SDKLocalIndex, error) {
	if err := m.ensureLocalLoad(false); err != nil {
		return nil, err
	}
	return m.localSDKs, nil
}

func (m *SDKManager) FindLocalSDK(name, version string) *models.InstalledSDK {
	_ = m.ensureLocalLoad(true)

	for i := range m.localSDKs.SDKs {
		sdk := &m.localSDKs.SDKs[i]
		if sdk.Name == name && sdk.Version == version {
			sdk.Index = i
			return sdk
		}
	}
	return nil
}

func (m *SDKManager) IndexLocalSDKs() error {
	if err := m.ensureLocalLoad(false); err != nil {
		return err
	}

	currentTime := time.Now()
	if m.localSDKs.CreatedAt.IsZero() {
		m.localSDKs.CreatedAt = currentTime
	}
	m.localSDKs.UpdatedAt = currentTime
	m.localSDKs.SDKs = nil
	clear(m.groupSDKs)

	for _, sdkCfg := range m.config.SDKs {
		ccolor.Cyanf("Starting find installed %q SDK\n", sdkCfg.Name)

		if sdkCfg.InstallDir != "" {
			ver2dirMap, err := xenvutil.ListVersionDirs(sdkCfg.InstallDir)
			if err != nil {
				return err
			}

			baseDir := filepath.Dir(sdkCfg.InstallDir)
			ccolor.Cyanf(" - from dir: %s\n", baseDir)
			for version, installPath := range ver2dirMap {
				ccolor.Infof("  Found %s %s\n", sdkCfg.Name, version)
				m.localSDKs.SDKs = append(m.localSDKs.SDKs, models.InstalledSDK{
					ID:         fmt.Sprintf("%s:%s", sdkCfg.Name, version),
					Name:       sdkCfg.Name,
					Version:    version,
					InstallDir: installPath,
					Source:     "xenv",
					CreatedAt:  currentTime,
					UpdatedAt:  currentTime,
				})
			}
		}

		if sdkCfg.OtherVersions != nil {
			for version, dirPath := range sdkCfg.OtherVersions {
				dirPath = fsutil.ExpandHome(dirPath)
				if !fsutil.IsDir(dirPath) {
					ccolor.Warnf("[W] Custum version %s path %q is not exists\n", version, dirPath)
					continue
				}

				ccolor.Infof("  Found %s %s(at %s)\n", sdkCfg.Name, version, dirPath)
				m.localSDKs.SDKs = append(m.localSDKs.SDKs, models.InstalledSDK{
					ID:         fmt.Sprintf("%s:%s", sdkCfg.Name, version),
					Name:       sdkCfg.Name,
					Version:    version,
					InstallDir: dirPath,
					Source:     "xenv",
					CreatedAt:  currentTime,
					UpdatedAt:  currentTime,
				})
			}
		}
	}

	ccolor.Magentaf("\nWrite indexed data to %s\n", m.localFile)
	return m.SaveLocalIndex()
}

func (m *SDKManager) SaveLocalIndex() error {
	if err := os.MkdirAll(filepath.Dir(m.localFile), 0o755); err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(m.localSDKs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.localFile, jsonBytes, 0o664)
}

func (m *SDKManager) AddSDK(name, version, installDir string) error {
	if err := m.ensureLocalLoad(false); err != nil {
		return err
	}

	currentTime := time.Now()
	if m.localSDKs.CreatedAt.IsZero() {
		m.localSDKs.CreatedAt = currentTime
	}
	m.localSDKs.UpdatedAt = currentTime
	m.localSDKs.SDKs = append(m.localSDKs.SDKs, models.InstalledSDK{
		ID:         fmt.Sprintf("%s:%s", name, version),
		Name:       name,
		Version:    version,
		InstallDir: installDir,
		Source:     "xenv",
		CreatedAt:  currentTime,
		UpdatedAt:  currentTime,
	})
	delete(m.groupSDKs, name)

	return m.SaveLocalIndex()
}

func (m *SDKManager) DeleteSDK(localSDK *models.InstalledSDK) error {
	if err := m.ensureLocalLoad(false); err != nil {
		return err
	}

	sdks := m.localSDKs.SDKs
	sdkIndex := localSDK.Index
	m.localSDKs.SDKs = append(sdks[:sdkIndex], sdks[sdkIndex+1:]...)
	delete(m.groupSDKs, localSDK.Name)

	return m.SaveLocalIndex()
}

func (m *SDKManager) FindSDKByID(id string) *models.InstalledSDK {
	_ = m.ensureLocalLoad(true)
	return m.localSDKs.FindByID(id)
}

func (m *SDKManager) ListSDKVersions(name string) []models.InstalledSDK {
	if ls, ok := m.groupSDKs[name]; ok {
		return ls
	}

	_ = m.ensureLocalLoad(true)
	ls := m.localSDKs.ListByName(name)
	if len(ls) > 0 {
		m.groupSDKs[name] = ls
	}
	return ls
}

func (m *SDKManager) ListMergedSDKVersions(name string) []models.InstalledSDK {
	localItems := m.ListSDKVersions(name)
	if m.config == nil || !m.config.EgetEnable {
		return localItems
	}

	egetItems, err := m.egetSrc.ListSDKVersions(name)
	if err != nil {
		ccolor.Warnf("WARN: failed to load eget SDK store %q: %v\n", m.egetSrc.Path, err)
		return localItems
	}
	if len(egetItems) == 0 {
		return localItems
	}

	merged := make(map[string]models.InstalledSDK, len(localItems)+len(egetItems))
	for _, item := range localItems {
		merged[item.Version] = item
	}
	for _, item := range egetItems {
		merged[item.Version] = item
	}

	items := make([]models.InstalledSDK, 0, len(merged))
	for _, item := range merged {
		items = append(items, item)
	}
	models.SortByVersionDesc(items)
	return items
}

func (m *SDKManager) MatchSDKByNameAndVersion(name, version string) *models.InstalledSDK {
	list := m.ListMergedSDKVersions(name)
	if len(list) == 0 {
		return nil
	}
	return m.MatchSDKByVersion(list, version)
}

func (m *SDKManager) MatchSDKByVersion(localSDKs []models.InstalledSDK, version string) *models.InstalledSDK {
	if len(localSDKs) == 0 {
		return nil
	}

	dotNum := strings.Count(version, ".")

	// 精确匹配优先，且不受末尾 '.' 段数限制，如同时存在 18 与 18.1 时，输入 18 返回 18
	for i := range localSDKs {
		if localSDKs[i].Version == version {
			return &localSDKs[i]
		}
	}

	if version == "latest" {
		// latest 取比较器意义上的最大版本，不依赖列表顺序
		latest := &localSDKs[0]
		for i := 1; i < len(localSDKs); i++ {
			if models.CompareVersionStrings(localSDKs[i].Version, latest.Version) > 0 {
				latest = &localSDKs[i]
			}
		}
		return latest
	}

	for i := range localSDKs {
		locVersion := localSDKs[i].Version
		if strings.HasPrefix(locVersion, version) {
			if len(locVersion) == len(version) || locVersion[len(version)] == '.' {
				return &localSDKs[i]
			}
		}
	}

	// 仅完整版本号（段数 > 2，如 1.23.1）才按档位向上匹配；
	// UpMatchNone(0) 及未知档位不做额外向上匹配。
	if dotNum > 1 && m.config != nil {
		parts := strings.Split(version, ".")

		switch m.config.AllowUpMatch {
		case xenvcom.UpMatchOne:
			// 同 minor 线内向上匹配，取高于输入的最小版本 eg: 1.23.1 -> 1.23.x
			if item := pickLowestHigherVersion(localSDKs, version, strings.Join(parts[:len(parts)-1], ".")+"."); item != nil {
				return item
			}
		case xenvcom.UpMatchTwo:
			// 同 major 线内向上匹配，取高于输入的最小版本 eg: 1.24.5 -> 1.24.x/1.25.x/1.26.x
			if item := pickLowestHigherVersion(localSDKs, version, parts[0]+"."); item != nil {
				return item
			}
		case xenvcom.UpMatchAll:
			// 允许任意更高的版本，取高于输入的最小版本
			if item := pickLowestHigherVersion(localSDKs, version, ""); item != nil {
				return item
			}
		}
	}

	return nil
}

// pickLowestHigherVersion 取数值上大于 version 的最小版本，prefix 非空时只考虑该前缀的版本。
// 传入列表通常按版本降序，故不能依赖列表顺序，必须逐个数值比较。
func pickLowestHigherVersion(localSDKs []models.InstalledSDK, version, prefix string) *models.InstalledSDK {
	var found *models.InstalledSDK
	for i := range localSDKs {
		item := &localSDKs[i]
		if prefix != "" && !strings.HasPrefix(item.Version, prefix) {
			continue
		}
		if models.CompareVersionStrings(item.Version, version) <= 0 {
			continue
		}
		if found == nil || models.CompareVersionStrings(item.Version, found.Version) < 0 {
			found = item
		}
	}
	return found
}

func (m *SDKManager) LocalIndex() *models.SDKLocalIndex {
	return m.localSDKs
}
