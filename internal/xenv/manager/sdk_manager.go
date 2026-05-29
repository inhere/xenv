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
	return nil
}

func (m *SDKManager) InitLoad() error {
	return m.ensureLocalLoad(false)
}

func (m *SDKManager) ensureLocalLoad(must bool) error {
	if m.localLoad {
		return nil
	}
	m.localLoad = true

	err := m.LoadLocalIndexIntoCache()
	if err != nil && must {
		panic(err)
	}
	return err
}

func (m *SDKManager) LoadLocalIndexIntoCache() error {
	if m.localFile == "" {
		m.localFile = fsutil.ExpandHome(xenvcom.InstalledMetaFile)
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

	for _, sdk := range m.localSDKs.SDKs {
		if sdk.Name == name && sdk.Version == version {
			return &sdk
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

func (m *SDKManager) MatchSDKByNameAndVersion(name, version string) *models.InstalledSDK {
	list := m.ListSDKVersions(name)
	if len(list) == 0 {
		return nil
	}
	return m.MatchSDKByVersion(list, version)
}

func (m *SDKManager) MatchSDKByVersion(localSDKs []models.InstalledSDK, version string) *models.InstalledSDK {
	dotNum := strings.Count(version, ".")

	if dotNum > 1 {
		for _, localSDK := range localSDKs {
			if localSDK.Version == version {
				return &localSDK
			}
		}
	}

	if version == "latest" {
		return &localSDKs[0]
	}

	for _, localSDK := range localSDKs {
		locVersion := localSDK.Version
		if strings.HasPrefix(locVersion, version) {
			if len(locVersion) == len(version) || locVersion[len(version)] == '.' {
				return &localSDK
			}
		}
	}

	if dotNum > 1 && m.config.AllowUpMatch > 0 {
		parts := strings.Split(version, ".")

		if m.config.AllowUpMatch == xenvcom.UpMatchOne {
			matchVer := strings.Join(parts[:len(parts)-1], ".") + "."
			for _, localSDK := range localSDKs {
				locVersion := localSDK.Version
				if strings.HasPrefix(locVersion, matchVer) {
					return &localSDK
				}
			}
		}
	}

	return nil
}

func (m *SDKManager) LocalIndex() *models.SDKLocalIndex {
	return m.localSDKs
}
