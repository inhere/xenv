package service

import (
	"path/filepath"

	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/shell"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
	"github.com/inhere/xenv/internal/xenv/xenvutil"
)

// getShellGenerator 获取当前shell的脚本生成器. 注意：不在shell hook环境，会返回nil
func getShellGenerator(_ *models.Configuration) (*shell.XenvScriptGenerator, error) {
	// hookShell 不为空表明在shell hook环境中
	hookShell := xenvcom.HookShell()
	if hookShell == "" {
		return nil, nil
	}

	shellType, err := shell.TypeFromString(hookShell)
	if err != nil {
		return nil, err
	}

	return shell.NewScriptGenerator(shellType), nil
}

func sdkVersionsFromSpecifiedFiles(specMap map[string]*models.VersionSpec) {
	sdkVersionsFromSpecifiedFilesInDir(specMap, xenvcom.SessionRootDir("."))
}

func sdkVersionsFromSpecifiedFilesInDir(specMap map[string]*models.VersionSpec, rootDir string) {
	// 支持识别常用的工具配置 eg: go.mod, .tool-versions, .nvmrc, .python-version
	toolsCfgFiles := []string{"go.work", "go.mod", ".tool-versions", ".nvmrc", ".python-version"}
	for _, filename := range toolsCfgFiles {
		filePath := filepath.Join(rootDir, filename)
		if !fsutil.IsFile(filePath) {
			continue
		}

		switch filename {
		case ".tool-versions":
			// 识别 .tool-versions 文件
			verMap, err := xenvutil.ParseToolVersions(filePath)
			if err != nil {
				ccolor.Warnf("Failed to parse .tool-versions file: %v\n", err)
				continue
			}

			ccolor.Infof("Detect tool versions from .tool-versions: %v\n", verMap)
			for name, ver := range verMap {
				specMap[name] = &models.VersionSpec{
					Name:    name,
					Version: ver,
				}
			}
		case "go.work", "go.mod":
			goVer, err := xenvutil.ParseGoVersion(filePath)
			if err != nil {
				ccolor.Warnf("Failed to parse go.mod file: %v\n", err)
				continue
			}
			ccolor.Infof("Detect go version from go.mod: %s\n", goVer)
			specMap["go"] = &models.VersionSpec{
				Name:    "go",
				Version: goVer,
			}
		case ".nvmrc":
			// 识别 .nvmrc 文件
			nodeVer, err := xenvutil.ParseNvmrcFile(filePath)
			if err != nil {
				ccolor.Warnf("Failed to parse .nvmrc file: %v\n", err)
				continue
			}
			ccolor.Infof("Detect node version from .nvmrc: %s\n", nodeVer)
			specMap["node"] = &models.VersionSpec{
				Name:    "node",
				Version: nodeVer,
			}
		case ".python-version":
			// 识别 .python-version 文件
			pyVer, err := xenvutil.ParsePythonVersion(filePath)
			if err != nil {
				ccolor.Warnf("Failed to parse .python-version file: %v\n", err)
				continue
			}
			ccolor.Infof("Detect python version from .python-version: %s\n", pyVer)
			specMap["python"] = &models.VersionSpec{
				Name:    "python",
				Version: pyVer,
			}
		}
	}
}
