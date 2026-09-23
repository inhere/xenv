package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/util"
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

	locals := ts.sdks.ListMergedSDKVersions(sdkCfg.Name)
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
	return util.NormalizePath(localSDK.InstallDir), nil
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

	script, params, err := ts.activateSDKs(gen, sdkSpecs, opFlag, true)
	if err != nil {
		return "", err
	}

	// direnv 作用域下变更会留在当前 shell, 需同步记录以便离开目录时恢复
	if script != "" && opFlag == models.OpFlagDirenv {
		script += mergeAppliedRecord(gen, deltaFromActivate(params))
	}
	return script, nil
}

// activateSDKs 激活 SDK 并返回本次应用的参数(用于记录可撤销变更)
func (ts *SDKService) activateSDKs(gen *shell.XenvScriptGenerator, sdkSpecs []*models.VersionSpec, opFlag models.OpFlag, saveState bool) (script string, actParams *models.ActivateSDKsParams, err error) {
	actParams = models.NewActivateSDKsParams()
	actParams.OpFlag = opFlag

	for _, spec := range sdkSpecs {
		localSDK, err3 := ts.checkActivateSDK(spec)
		if err3 != nil {
			return "", actParams, fmt.Errorf("failed to activate sdk %q: %w", spec, err3)
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
		ts.warnTemporaryRuntimeOverride(spec, opFlag)
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

	if !saveState {
		return sb.String(), actParams, nil
	}
	if !isEmpty {
		ts.state.SetBatchMode(true)
		defer ts.state.SetBatchMode(false)

		err = ts.state.UseSDKsWithParams(actParams)
		if err != nil {
			return "", actParams, err
		}
		err = ts.state.SaveStateFile()
		return sb.String(), actParams, err
	}
	return "", actParams, nil
}

func (ts *SDKService) checkActivateSDK(spec *models.VersionSpec) (*models.InstalledSDK, error) {
	sdkCfg := ts.config.FindSDKConfig(spec.Name)
	if sdkCfg == nil {
		return nil, fmt.Errorf("sdk %s config is not defined", spec.Name)
	}
	spec.Name = sdkCfg.Name

	localSDKs := ts.sdks.ListSDKVersions(sdkCfg.Name)
	localSDK := ts.sdks.MatchSDKByVersion(localSDKs, spec.Version)
	if localSDK == nil && ts.config.EgetEnable {
		localSDK = ts.sdks.MatchSDKByVersion(ts.sdks.ListMergedSDKVersions(sdkCfg.Name), spec.Version)
	}
	if localSDK == nil {
		if isUnsupportedVersionAlias(spec.Version) {
			return nil, fmt.Errorf("sdk version alias %q is not supported yet, use an explicit version", spec.Version)
		}
		return nil, fmt.Errorf("sdk %s is not installed locally", spec.ID())
	}

	localSDK.Config = sdkCfg
	spec.RealVersion = localSDK.Version
	return localSDK, nil
}

// isUnsupportedVersionAlias 判断是否为未支持的非数字版本别名 eg: lts, auto
func isUnsupportedVersionAlias(version string) bool {
	switch version {
	case "latest", "stable":
		return false
	}
	return !strings.ContainsAny(version, "0123456789")
}

func (ts *SDKService) warnTemporaryRuntimeOverride(spec *models.VersionSpec, opFlag models.OpFlag) {
	if opFlag != models.OpFlagSession {
		return
	}

	deState := ts.state.Nearest()
	if deState == nil || deState.IsEmpty() {
		return
	}

	dirVersion := models.FilterSDKsForGOOS(deState.SDKs, runtime.GOOS)[spec.Name]
	if dirVersion == "" || dirVersion == spec.Version || dirVersion == spec.RealVersion {
		return
	}

	dirSpec := &models.VersionSpec{Name: spec.Name, Version: dirVersion}
	if dirSDK, err := ts.checkActivateSDK(dirSpec); err == nil && dirSDK.Version == spec.RealVersion {
		return
	}

	ccolor.Warnf(
		"WARN: directory state wants %s:%s; this activation is a temporary runtime override. Use `xenv use -s %s:%s` to update .xenv.toml.\n",
		spec.Name,
		dirVersion,
		spec.Name,
		spec.Version,
	)
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

	deState := ts.state.Nearest()
	dirFile := ""
	if deState != nil && !deState.IsEmpty() {
		dirFile = deState.File
	}

	// 同一项目(含子目录)已应用过: 不重复应用, 也不撤销
	applied := loadAppliedRecord()
	if applied != nil && dirFile != "" && applied.File == dirFile {
		return "", nil
	}

	var sb strutil.Builder
	if applied != nil {
		sb.WriteString(buildLeaveScript(gen, applied))

		// 应用脚本必须基于撤销后的环境计算, 否则会把刚移除的 PATH 条目再加回来
		restoreEnv := simulateLeaveInProcess(applied)
		defer restoreEnv()
	}

	script, rec, err := ts.applyDirenvState(gen, deState)
	if err != nil {
		return "", err
	}
	sb.WriteString(script)

	// 记录随脚本写入 shell 环境, 供下一次 cd 读取
	sb.WriteString(writeAppliedRecord(gen, rec))

	if sb.Len() > 0 {
		return sb.String(), nil
	}
	return "", nil
}

// applyDirenvState 应用当前目录的 direnv 状态, 并返回本次的可撤销变更
func (ts *SDKService) applyDirenvState(gen *shell.XenvScriptGenerator, deState *models.ActivityState) (string, *models.AppliedDirenv, error) {
	var specMap = make(map[string]*models.VersionSpec)
	opFlag := models.OpFlagSession

	if deState != nil && !deState.IsEmpty() {
		opFlag = models.OpFlagDirenv
		xenvcom.Debugf("Detect xenv state file: %s\n", deState.File)
		for name, ver := range models.FilterSDKsForGOOS(deState.SDKs, runtime.GOOS) {
			specMap[name] = &models.VersionSpec{
				Name:    name,
				Version: ver,
			}
		}
	} else {
		sdkVersionsFromSpecifiedFiles(specMap)
	}

	var sb strutil.Builder
	var actParams *models.ActivateSDKsParams
	if len(specMap) > 0 {
		sdkSpecs := make([]*models.VersionSpec, 0, len(specMap))
		for _, spec := range specMap {
			sdkSpecs = append(sdkSpecs, spec)
		}
		saveState := opFlag != models.OpFlagDirenv || deState == nil
		script, params, err := ts.activateSDKs(gen, sdkSpecs, opFlag, saveState)
		if err != nil {
			return "", nil, err
		}
		sb.WriteString(script)
		actParams = params
	}

	var dirPaths []string
	if opFlag == models.OpFlagDirenv && deState != nil {
		dirPaths = models.FilterPathsForGOOS(deState.Paths, runtime.GOOS)
		if len(dirPaths) > 0 {
			sb.WriteString(gen.GenAddPaths(dirPaths))
		}
		if len(deState.Envs) > 0 {
			sb.WriteString(gen.GenSetEnvs(deState.Envs))
		}
	}

	if projectScript := ts.genProjectScriptForDirenv(gen, deState); projectScript != "" {
		sb.WriteString(projectScript)
	}
	if envrcScript := ts.genEnvrcScript(gen); envrcScript != "" {
		sb.WriteString(envrcScript)
	}

	// 配置开启时, 进入目录顺带检查 [tools] 要求(只提示, 不阻断)
	for _, warning := range ts.direnvToolWarnings(deState) {
		ccolor.Warnf("WARN: %s\n", warning)
	}

	if opFlag != models.OpFlagDirenv || deState == nil {
		// 非 .xenv.toml 驱动的激活(如按 go.mod 自动检测)按会话默认值处理, 不记录撤销
		return sb.String(), nil, nil
	}

	rec := models.NewAppliedDirenv(deState.File)
	if actParams != nil {
		for _, path := range actParams.AddPaths {
			if !inSessionPath(path) {
				rec.AddAppliedPath(path)
			}
		}
		for name := range actParams.AddEnvs {
			name = strings.ToUpper(name)
			prev := os.Getenv(name)
			rec.AddAppliedEnv(name, prev, prev != "")
		}
		for name, version := range actParams.AddSdks {
			rec.AddAppliedSDK(name, version)
		}
	}
	for _, path := range dirPaths {
		normalized := util.NormalizePath(path)
		if !inSessionPath(normalized) {
			rec.AddAppliedPath(normalized)
		}
	}
	for name := range deState.Envs {
		name = strings.ToUpper(name)
		prev := os.Getenv(name)
		rec.AddAppliedEnv(name, prev, prev != "")
	}
	return sb.String(), rec, nil
}

// buildLeaveScript 生成撤销上一次 direnv 应用的脚本
func buildLeaveScript(gen *shell.XenvScriptGenerator, rec *models.AppliedDirenv) string {
	var sb strings.Builder

	// 1. 移除本次应用加入的 PATH 条目
	if len(rec.Paths) > 0 {
		pathList := sessionPath()
		for _, path := range rec.Paths {
			pathList, _ = withoutPath(pathList, path)
		}
		sb.WriteString(gen.GenSetPath(pathList))
	}

	// 2. 恢复变量旧值, 应用前未设置的则取消设置
	for _, item := range rec.Envs {
		if item.HadPrev {
			sb.WriteString(gen.GenSetEnv(item.Name, item.Prev))
			continue
		}
		sb.WriteString(gen.GenUnsetEnv(item.Name))
	}
	return sb.String()
}

// simulateLeaveInProcess 在进程内临时模拟撤销结果, 供应用脚本基于撤销后的环境计算
//
// 返回的函数用于恢复进程环境
func simulateLeaveInProcess(rec *models.AppliedDirenv) func() {
	var restores []func()
	restoreEnv := func(name string, value string, had bool) {
		restores = append(restores, func() {
			if had {
				_ = os.Setenv(name, value)
				return
			}
			_ = os.Unsetenv(name)
		})
	}

	if len(rec.Paths) > 0 {
		oldPath, hadPath := os.LookupEnv("PATH")
		pathList := sessionPath()
		for _, path := range rec.Paths {
			pathList, _ = withoutPath(pathList, path)
		}
		_ = os.Setenv("PATH", strings.Join(pathList, xenvcom.PathSep()))
		restoreEnv("PATH", oldPath, hadPath)
	}

	for _, item := range rec.Envs {
		old, had := os.LookupEnv(item.Name)
		if item.HadPrev {
			_ = os.Setenv(item.Name, item.Prev)
		} else {
			_ = os.Unsetenv(item.Name)
		}
		restoreEnv(item.Name, old, had)
	}

	return func() {
		for i := len(restores) - 1; i >= 0; i-- {
			restores[i]()
		}
	}
}

// inSessionPath 检查路径是否已在当前 shell 会话的 PATH 中
func inSessionPath(path string) bool {
	_, found := withoutPath(sessionPath(), path)
	return found
}

// direnvToolWarnings 返回 direnv 状态中工具要求的告警
//
// 进入目录时只检查是否存在, 版本比较仍由 `xenv check tools` 完成
func (ts *SDKService) direnvToolWarnings(deState *models.ActivityState) []string {
	if !ts.config.CheckToolsOnDirenv || deState == nil || len(deState.ToolRequirements) == 0 {
		return nil
	}

	var warnings []string
	for _, result := range NewCheckService(ts).CheckTools(deState, false) {
		if result.Status == CheckStatusOK {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("tool %s: %s", result.Name, result.Message))
	}
	return warnings
}

// genEnvrcScript 生成 source .envrc / .envrc.ps1 的脚本
//
// 与项目脚本一致, 仅在 source_project_scripts 开启时生效
func (ts *SDKService) genEnvrcScript(gen *shell.XenvScriptGenerator) string {
	if !ts.config.SourceProjectScripts {
		return ""
	}

	var sb strings.Builder
	for _, filePath := range ts.state.EnvrcFiles() {
		sb.WriteString(gen.GenSourceFile(filePath))
	}
	return sb.String()
}

func (ts *SDKService) WriteHookToProfile(st shell.ShType, pwshProfile string) error {
	if xenvcom.InHookShell() {
		ccolor.Infoln("The hook script is already installed in the current shell")
		return nil
	}

	profilePath, err := shell.NewScriptGenerator(st).InstallToProfile(pwshProfile)
	if err != nil {
		return err
	}

	ccolor.Infof("Installed xenv hook to: %s\n", profilePath)
	return nil
}

func (ts *SDKService) GenHookScripts(st shell.ShType) (string, error) {
	gen := shell.NewScriptGenerator(st)
	if err := ts.sdks.InitLoad(); err != nil {
		return "", err
	}

	state := ts.state.Merged()
	params := &models.GenInitScriptParams{
		Envs:  ts.config.GlobalEnv,
		Paths: models.FilterPathsForGOOS(ts.config.GlobalPaths, runtime.GOOS),
	}
	params.AddPaths(models.FilterPathsForGOOS(state.Paths, runtime.GOOS))
	params.Envs = maputil.AppendSMap(params.Envs, state.Envs)
	params.ShellAliases = ts.config.ShellAliases
	params.ShellHooksDir = ts.config.ShellHooksDir

	sdks := models.FilterSDKsForGOOS(state.SDKs, runtime.GOOS)
	if len(sdks) > 0 {
		for name, version := range sdks {
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

func (ts *SDKService) genProjectScriptForDirenv(gen *shell.XenvScriptGenerator, deState *models.ActivityState) string {
	if !ts.config.SourceProjectScripts || deState == nil || deState.File == "" {
		return ""
	}

	projectDir := filepath.Dir(deState.File)
	switch xenvcom.HookShell() {
	case "bash", "zsh":
		if _, err := os.Stat(filepath.Join(projectDir, ".xenv.sh")); err == nil {
			return gen.GenSourceProjectScript(projectDir)
		}
	case "pwsh":
		if _, err := os.Stat(filepath.Join(projectDir, ".xenv.ps1")); err == nil {
			return gen.GenSourceProjectScript(projectDir)
		}
	}
	return ""
}

func (ts *SDKService) DeactivateSDKs(deSDKs []string, opFlag models.OpFlag) (script string, err error) {
	ts.state.SetBatchMode(true)
	defer ts.state.SetBatchMode(false)

	gen, err1 := getShellGenerator(ts.config)
	if err1 != nil {
		return "", err1
	}

	var delPaths, delEnvs []string
	var delSDKNames []string

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
		if !ts.isSDKActiveForDeactivate(spec.Name, opFlag) {
			continue
		}
		delSDKNames = append(delSDKNames, spec.Name)

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

	err = ts.state.DelSDKsWithEnvsPaths(delSDKNames, delEnvs, delPaths, opFlag)
	if err != nil {
		return "", err
	}

	err = ts.state.SaveStateFile()
	return sb.String(), err
}

func (ts *SDKService) isSDKActiveForDeactivate(name string, opFlag models.OpFlag) bool {
	switch opFlag {
	case models.OpFlagGlobal:
		return ts.state.Global().SDKs[name] != ""
	case models.OpFlagDirenv:
		ds := ts.state.Nearest()
		return ds != nil && ds.SDKs[name] != ""
	default:
		return ts.state.Merged().SDKs[name] != ""
	}
}

func (ts *SDKService) checkDeactivateSDK(spec *models.VersionSpec, opFlag models.OpFlag) (*models.InstalledSDK, error) {
	sdkCfg := ts.config.FindSDKConfig(spec.Name)
	if sdkCfg == nil {
		return nil, fmt.Errorf("sdk %s config is not defined", spec.Name)
	}
	spec.Name = sdkCfg.Name

	localSDKs := ts.sdks.ListMergedSDKVersions(sdkCfg.Name)
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
