package service

import (
	"fmt"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv/manager"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/sdk"
	"github.com/inhere/xenv/internal/xenv/shell"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

type SDKService struct {
	config *models.Configuration
	state  *manager.StateManager
	sdks   *manager.SDKManager
}

func NewSDKService(config *models.Configuration, state *manager.StateManager, sdkMgr *manager.SDKManager) *SDKService {
	return &SDKService{
		config: config,
		state:  state,
		sdks:   sdkMgr,
	}
}

func (ts *SDKService) IndexLocalSDKs() error {
	return ts.sdks.IndexLocalSDKs()
}

func (ts *SDKService) ListSDKs(showAll bool) error {
	cfgSdks := ts.config.SDKs
	if len(cfgSdks) == 0 {
		fmt.Println("No SDKs configured for management. see config: sdks")
		return nil
	}
	if err := ts.sdks.InitLoad(); err != nil {
		return err
	}

	ccolor.Cyanf("Managed SDKs(%d):\n", len(cfgSdks))
	for _, sdkCfg := range cfgSdks {
		ccolor.Magentaf(" %s", sdkCfg.Name)
		if len(sdkCfg.Alias) > 0 {
			fmt.Printf("(Alias: %v):\n", sdkCfg.Alias)
		} else {
			fmt.Println(":")
		}
		fmt.Printf("  - InstallDir: %s\n", sdkCfg.InstallDir)

		locals := ts.sdks.ListMergedSDKVersions(sdkCfg.Name)
		if len(locals) == 0 {
			if !showAll {
				continue
			}
			fmt.Print("  - Installed: ")
			ccolor.Cyanln("None")
			continue
		}

		fmt.Println("  - Installed:")
		for _, local := range locals {
			ccolor.Infof("    %s  %s  %s\n", local.Version, local.Source, local.InstallDir)
		}
	}
	return nil
}

func (ts *SDKService) ShowSDK(name string) error {
	sdkCfg := ts.config.FindSDKConfig(name)
	if sdkCfg == nil {
		return fmt.Errorf("sdk %s is not configured", name)
	}

	fmt.Printf("SDK: %s\n", sdkCfg.Name)
	if sdkCfg.Alias != "" {
		fmt.Printf("  Alias: %s\n", sdkCfg.Alias)
	}
	fmt.Printf("  InstallDir: %s\n", sdkCfg.InstallDir)
	if sdkCfg.BinDir != "" {
		fmt.Printf("  BinDir: %s\n", sdkCfg.BinDir)
	}

	locals := ts.sdks.ListMergedSDKVersions(name)
	if len(locals) == 0 {
		fmt.Println("  Installed: none")
		return nil
	}

	fmt.Println("  Installed:")
	for _, local := range locals {
		fmt.Printf("    %s  %s  %s\n", local.Version, local.Source, local.InstallDir)
	}
	return nil
}

func (ts *SDKService) WhereSDK(spec string, bin bool) (string, error) {
	sdkSpec, err := sdk.ParseVersionSpec(spec)
	if err != nil {
		return "", err
	}

	localSDK, err := ts.checkActivateSDK(sdkSpec)
	if err != nil {
		return "", err
	}
	if bin {
		return localSDK.BinDirPath(), nil
	}
	return localSDK.InstallDir, nil
}

func (ts *SDKService) ActivateSDKs(useSDKs []string, opFlag models.OpFlag) (script string, err error) {
	var sdkSpecs []*models.VersionSpec
	for _, arg := range useSDKs {
		spec, err2 := sdk.ParseVersionSpec(arg)
		if err2 != nil {
			return "", err2
		}
		sdkSpecs = append(sdkSpecs, spec)
	}

	gen, err1 := getShellGenerator(ts.config)
	if err1 != nil {
		return "", err1
	}

	return ts.activateSDKs(gen, sdkSpecs, opFlag)
}

func (ts *SDKService) activateSDKs(gen *shell.XenvScriptGenerator, sdkSpecs []*models.VersionSpec, opFlag models.OpFlag) (script string, err error) {
	actParams := models.NewActivateSDKsParams()
	actParams.OpFlag = opFlag

	for _, spec := range sdkSpecs {
		localSDK, err3 := ts.checkActivateSDK(spec)
		if err3 != nil {
			return "", fmt.Errorf("failed to activate sdk %q: %w", spec, err3)
		}

		oldActiveVer := ts.state.Merged().SDKs[spec.Name]
		if oldActiveVer != "" {
			oldSDK := ts.sdks.MatchSDKByNameAndVersion(spec.Name, oldActiveVer)
			if oldSDK != nil {
				oldSDK.Config = localSDK.Config
				actParams.AddRemPath(oldSDK.BinDirPath())
			}
		}

		actParams.AddSdk(spec.Name, localSDK.Version)
		actParams.AddPath(localSDK.BinDirPath())
		if len(localSDK.Config.ActiveEnv) > 0 {
			actParams.AddSetEnvs(localSDK.RenderActiveEnv())
		}

		if opFlag == models.OpFlagGlobal {
			ccolor.Infof("Activate %s for global default\n", localSDK.ID)
		} else if opFlag == models.OpFlagDirenv {
			ccolor.Infof("Activate %s for direnv state\n", localSDK.ID)
		} else {
			ccolor.Infof("Activate %s for current session\n", localSDK.ID)
		}
	}

	var sb strutil.Builder
	isEmpty := actParams.IsEmpty()
	if gen != nil && !isEmpty {
		sb.WriteString(gen.GenRemThenAddPaths(actParams.RemPaths, actParams.AddPaths))
		if len(actParams.AddEnvs) > 0 {
			sb.WriteString(gen.GenSetEnvs(actParams.AddEnvs))
		}
	}
	if gen == nil {
		ccolor.Warnln("TIP: The operation will not take effect, please setup the SHELL HOOK first.")
	}

	if !isEmpty {
		ts.state.SetBatchMode(true)
		defer ts.state.SetBatchMode(false)

		err = ts.state.UseSDKsWithParams(actParams)
		if err != nil {
			return "", err
		}
		err = ts.state.SaveStateFile()
		return sb.String(), err
	}
	return "", nil
}

func (ts *SDKService) checkActivateSDK(spec *models.VersionSpec) (*models.InstalledSDK, error) {
	sdkCfg := ts.config.FindSDKConfig(spec.Name)
	if sdkCfg == nil {
		return nil, fmt.Errorf("sdk %s config is not defined", spec.Name)
	}

	localSDKs := ts.sdks.ListSDKVersions(sdkCfg.Name)
	if len(localSDKs) == 0 {
		return nil, fmt.Errorf("sdk %s is not installed locally", spec.Name)
	}

	localSDK := ts.sdks.MatchSDKByVersion(localSDKs, spec.Version)
	if localSDK == nil {
		return nil, fmt.Errorf("sdk %s is not installed locally", spec.ID())
	}

	localSDK.Config = sdkCfg
	spec.RealVersion = localSDK.Version
	return localSDK, nil
}

func (ts *SDKService) SetupDirenv() (string, error) {
	gen, err := getShellGenerator(ts.config)
	if err != nil {
		return "", err
	}
	if gen == nil {
		ccolor.Warnf("TIP: The operation will not take effect, please setup the SHELL HOOK first.")
		return "", nil
	}

	var specMap = make(map[string]*models.VersionSpec)
	opFlag := models.OpFlagSession

	deState := ts.state.Nearest()
	if deState != nil && !deState.IsEmpty() {
		opFlag = models.OpFlagDirenv
		ccolor.Infof("Detect xenv state file: %s\n", deState.File)
		for name, ver := range deState.SDKs {
			specMap[name] = &models.VersionSpec{
				Name:    name,
				Version: ver,
			}
		}
	} else {
		sdkVersionsFromSpecifiedFiles(specMap)
	}

	if len(specMap) > 0 {
		sdkSpecs := make([]*models.VersionSpec, 0, len(specMap))
		for _, spec := range specMap {
			sdkSpecs = append(sdkSpecs, spec)
		}
		return ts.activateSDKs(gen, sdkSpecs, opFlag)
	}
	return "", nil
}

func (ts *SDKService) WriteHookToProfile(st shell.ShType, pwshProfile string) error {
	gen := shell.NewScriptGenerator(st)
	if xenvcom.InHookShell() {
		ccolor.Infoln("The hook script is already installed in the current shell")
		return nil
	}

	return gen.InstallToProfile(pwshProfile)
}

func (ts *SDKService) GenHookScripts(st shell.ShType) (string, error) {
	gen := shell.NewScriptGenerator(st)
	if err := ts.sdks.InitLoad(); err != nil {
		return "", err
	}

	state := ts.state.Merged()
	params := &models.GenInitScriptParams{
		Envs:  ts.config.GlobalEnv,
		Paths: ts.config.GlobalPaths,
	}
	params.AddPaths(state.Paths)
	params.Envs = maputil.AppendSMap(params.Envs, state.Envs)
	params.ShellAliases = ts.config.ShellAliases
	params.ShellHooksDir = ts.config.ShellHooksDir

	if len(state.SDKs) > 0 {
		for name, version := range state.SDKs {
			spec := &models.VersionSpec{Name: name, Version: version}
			localSDK, err := ts.checkActivateSDK(spec)
			if err != nil {
				continue
			}
			params.AddPath(localSDK.BinDirPath())
		}
	}

	return gen.GenHookScripts(params)
}

func (ts *SDKService) DeactivateSDKs(deSDKs []string, opFlag models.OpFlag) (script string, err error) {
	ts.state.SetBatchMode(true)
	defer ts.state.SetBatchMode(false)

	gen, err1 := getShellGenerator(ts.config)
	if err1 != nil {
		return "", err1
	}

	var delPaths, delEnvs []string

	for _, arg := range deSDKs {
		spec, err2 := sdk.ParseVersionSpec(arg)
		if err2 != nil {
			return "", err2
		}

		localSDK, err3 := ts.checkDeactivateSDK(spec, opFlag)
		if err3 != nil {
			ccolor.Warnf("WARN: failed to deactivate sdk %q: %v", spec, err3)
			continue
		}

		if localSDK != nil {
			delPaths = append(delPaths, localSDK.BinDirPath())
			if len(localSDK.Config.ActiveEnv) > 0 {
				delEnvs = append(delEnvs, localSDK.Config.ActiveEnvNames()...)
			}
		}

		if opFlag == models.OpFlagGlobal {
			ccolor.Infof("Deactivate %s for global default\n", spec)
		} else if opFlag == models.OpFlagDirenv {
			ccolor.Infof("Deactivate %s for direnv state\n", spec)
		} else {
			ccolor.Infof("Deactivate %s for current session\n", spec)
		}
	}

	var sb strutil.Builder
	if gen != nil && len(delPaths) > 0 {
		script1, notFounds := gen.GenRemovePaths(delPaths)
		if len(notFounds) > 0 {
			ccolor.Warnf("WARN: %d paths not found in PATH: %v\n", len(notFounds), notFounds)
		}

		sb.Writeln(script1)
		if len(delEnvs) > 0 {
			sb.Writeln(gen.GenUnsetEnvs(delEnvs))
		}
	}

	err = ts.state.DelSDKsWithEnvsPaths(deSDKs, delEnvs, delPaths, opFlag)
	if err != nil {
		return "", err
	}

	err = ts.state.SaveStateFile()
	return sb.String(), err
}

func (ts *SDKService) checkDeactivateSDK(spec *models.VersionSpec, opFlag models.OpFlag) (*models.InstalledSDK, error) {
	sdkCfg := ts.config.FindSDKConfig(spec.Name)
	if sdkCfg == nil {
		return nil, fmt.Errorf("sdk %s config is not defined", spec.Name)
	}

	localSDKs := ts.sdks.ListSDKVersions(sdkCfg.Name)
	if len(localSDKs) == 0 {
		return nil, fmt.Errorf("sdk %s is not installed locally", spec.Name)
	}

	localSDK := ts.sdks.MatchSDKByVersion(localSDKs, spec.Version)
	if localSDK == nil {
		return nil, nil
	}

	localSDK.Config = sdkCfg
	return localSDK, nil
}
