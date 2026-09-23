package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/util"
	"github.com/inhere/xenv/internal/xenv/manager"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/sysenv"
)

// EnvService handles environment variable and PATH management
type EnvService struct {
	config *models.Configuration
	state  *manager.StateManager
	// envMgr *manager.EnvManager
}

// NewEnvService creates a new EnvService
func NewEnvService(config *models.Configuration, state *manager.StateManager) *EnvService {
	return &EnvService{
		config: config,
		state:  state,
		// envMgr: manager.NewEnvManager(),
	}
}

func (s *EnvService) GlobalState() *models.ActivityState {
	return s.state.Global()
}

func (s *EnvService) SessionState() *models.ActivityState {
	return s.state.Session()
}

// endregion
// region ENV management
//

// SetEnvs sets multiple environment variables, envs is a list of KEY=VALUE
func (s *EnvService) SetEnvs(envs []string, opFlag models.OpFlag) (script string, err error) {
	envMap, err := parseEnvMap(envs)
	if err != nil {
		return "", err
	}

	// Generate shell eval scripts
	gen, err1 := getShellGenerator(s.config)
	if err1 != nil {
		return "", err1
	}

	// 在shell hook环境中, 按名称排序生成脚本, 保证输出稳定
	if gen != nil {
		var sb strings.Builder
		for _, name := range sortedNames(envMap) {
			sb.WriteString(gen.GenSetEnv(name, envMap[name]))
		}
		script = sb.String()
	} else {
		ccolor.Warnln("TIP: The operation will not take effect, please setup the SHELL HOOK first.")
	}

	// Add to activity state data
	err = s.state.AddEnvs(envMap, opFlag)
	return
}

// UnsetEnvs unsets multi environment variables
func (s *EnvService) UnsetEnvs(names []string, opFlag models.OpFlag) (script string, err error) {
	var sb strings.Builder
	// Generate shell eval scripts
	gen, err1 := getShellGenerator(s.config)
	if err1 != nil {
		return "", err1
	}
	if gen == nil {
		ccolor.Warnln("TIP: The operation will not take effect, please setup the SHELL HOOK first.")
	}

	s.state.SetBatchMode(true)
	defer s.state.SetBatchMode(false)
	for _, name := range names {
		name = strings.ToUpper(name)
		if val := os.Getenv(name); val == "" {
			ccolor.Warnf("ENV var not found: %s\n", name)
		}

		// 在shell hook环境中, 生成ENV set脚本
		if gen != nil {
			sb.WriteString(gen.GenUnsetEnv(name))
		}

		err = s.state.UnsetEnv(name, opFlag)
		if err != nil {
			return "", err
		}
	}

	err = s.state.SaveStateFile()
	return sb.String(), err
}

// GlobalEnv lists environment variables in the global scope
func (s *EnvService) GlobalEnv() map[string]string {
	// Return the global environment variables
	return maputil.MergeStrMap(s.config.GlobalEnv, s.state.Global().Envs)
}

// SessionEnv lists environment variables in the current session
func (s *EnvService) SessionEnv() map[string]string {
	return s.state.Session().Envs
}

// endregion
// region PATH management
//

// AddPath adds a path to the PATH environment variable
func (s *EnvService) AddPath(path string, opFlag models.OpFlag) (script string, err error) {
	normalizedPath := util.NormalizePath(path)

	// Check if path exists
	if _, err := os.Stat(normalizedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", normalizedPath)
	}

	// Add to session PATH
	currentPath := os.Getenv("PATH")
	pathList := util.SplitPath(currentPath)

	// Check if path already exists
	for _, p := range pathList {
		if p == normalizedPath {
			return "", fmt.Errorf("path already exists in PATH: %s", normalizedPath)
		}
	}

	// Generate shell eval scripts
	gen, err1 := getShellGenerator(s.config)
	if err1 != nil {
		return "", err1
	}

	// Add the path to the beginning of PATH (highest priority)
	// newPathList := append([]string{normalizedPath}, pathList...)

	// 在shell hook环境中, 生成 ENV set 脚本
	if gen != nil {
		script = gen.GenAddPath(normalizedPath)
	} else {
		ccolor.Warnln("TIP: The operation will not take effect, please setup the SHELL HOOK first.")
	}

	// Add to activity state
	err = s.state.AddPath(normalizedPath, opFlag)
	return
}

// RemovePath removes a path from the PATH environment variable
func (s *EnvService) RemovePath(path string, opFlag models.OpFlag) (script string, err error) {
	// Normalize the path
	normalizedPath := util.NormalizePath(path)
	pathList := util.SplitPath(os.Getenv("PATH"))

	found := false
	var newPaths []string

	// Remove from session PATH
	for _, p := range pathList {
		if p != normalizedPath {
			newPaths = append(newPaths, p)
		} else {
			found = true
		}
	}
	if !found {
		ccolor.Warnf("path not found in PATH: %s", normalizedPath)
		return "", fmt.Errorf("path not found in PATH: %s", normalizedPath)
	}

	// Generate shell eval scripts
	gen, err1 := getShellGenerator(s.config)
	if err1 != nil {
		return "", err1
	}

	// 在shell hook环境中, 生成 ENV set 脚本
	if gen != nil {
		script = gen.GenSetPath(newPaths)
	} else {
		ccolor.Warnln("TIP: The operation will not take effect, please setup the SHELL HOOK first.")
	}

	// Remove from activity state
	err = s.state.DelPath(normalizedPath, opFlag)
	return
}

// ListPaths lists PATH entries
func (s *EnvService) ListPaths() []models.PathEntry {
	var paths []models.PathEntry
	for _, entry := range s.state.Global().Paths {
		paths = append(paths, models.PathEntry{
			Path:     entry,
			Priority: 0,
			IsActive: true,
			Scope:    "global",
		})
	}

	for _, entry := range s.state.Session().Paths {
		paths = append(paths, models.PathEntry{
			Path:     entry,
			Priority: 0,
			IsActive: true,
			Scope:    "session",
		})
	}
	return paths
}

// SearchPath searches for a path in PATH
func (s *EnvService) SearchPath(path string) []string {
	normalizedPath := util.NormalizePath(path)
	var matches []string

	// Search in active paths
	for _, p := range s.state.Global().Paths {
		if strings.Contains(p, normalizedPath) {
			matches = append(matches, p)
		}
	}

	// Also search in current system PATH
	currentPath := os.Getenv("PATH")
	pathList := util.SplitPath(currentPath)
	for _, p := range pathList {
		if strings.Contains(p, normalizedPath) {
			matches = append(matches, p)
		}
	}

	return matches
}

// endregion
// region OS system ENV management
//

// SetSystemEnvs 批量设置操作系统的用户级环境变量, envs 为 KEY=VALUE 列表
func (s *EnvService) SetSystemEnvs(envs []string) (script string, err error) {
	envMap, err := parseEnvMap(envs)
	if err != nil {
		return "", err
	}

	// 逐个写入, 前一个失败时后面的不再处理
	for _, name := range sortedNames(envMap) {
		if err = sysenv.SetVar(name, envMap[name]); err != nil {
			return "", err
		}
	}

	// 在shell hook环境中, 按名称排序生成脚本让当前 shell 立即生效
	gen, err := getShellGenerator(s.config)
	if err != nil {
		return "", err
	}
	if gen != nil {
		var sb strings.Builder
		for _, name := range sortedNames(envMap) {
			sb.WriteString(gen.GenSetEnv(name, envMap[name]))
		}
		script = sb.String()
	}
	return script, nil
}

// UnsetSystemEnvs 删除操作系统的用户级环境变量
func (s *EnvService) UnsetSystemEnvs(names []string) (script string, err error) {
	gen, err := getShellGenerator(s.config)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, name := range names {
		name, err = normalizeEnvName(name)
		if err != nil {
			return "", err
		}
		if err = sysenv.UnsetVar(name); err != nil {
			return "", err
		}

		if gen != nil {
			sb.WriteString(gen.GenUnsetEnv(name))
		}
	}
	return sb.String(), nil
}

// AddSystemPath 添加路径到操作系统的用户级 PATH
func (s *EnvService) AddSystemPath(path string) (script string, err error) {
	normalizedPath := util.NormalizePath(path)
	if _, err = os.Stat(normalizedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", normalizedPath)
	}

	if err = sysenv.AddPath(normalizedPath); err != nil {
		return "", err
	}

	// 在shell hook环境中, 生成脚本让当前 shell 立即生效
	gen, err := getShellGenerator(s.config)
	if err != nil {
		return "", err
	}
	if gen != nil {
		if _, found := withoutPath(sessionPath(), normalizedPath); !found {
			script = gen.GenAddPath(normalizedPath)
		}
	}
	return script, nil
}

// RemoveSystemPath 删除操作系统的用户级 PATH 中的路径
func (s *EnvService) RemoveSystemPath(path string) (script string, err error) {
	normalizedPath := util.NormalizePath(path)

	if err = sysenv.RemovePath(normalizedPath); err != nil {
		return "", err
	}

	// 在shell hook环境中, 生成脚本移除当前 shell 的 PATH 条目
	gen, err := getShellGenerator(s.config)
	if err != nil {
		return "", err
	}
	if gen != nil {
		if newPaths, found := withoutPath(sessionPath(), normalizedPath); found {
			script = gen.GenSetPath(newPaths)
		}
	}
	return script, nil
}

// sessionPath 返回当前 shell 会话的 PATH 条目
func sessionPath() []string {
	return util.SplitPath(os.Getenv("PATH"))
}

// withoutPath 删除列表中的指定路径, 第二个返回值表示是否找到并删除
func withoutPath(pathList []string, path string) ([]string, bool) {
	newPaths := make([]string, 0, len(pathList))
	found := false
	for _, p := range pathList {
		if util.NormalizePath(p) == path {
			found = true
			continue
		}
		newPaths = append(newPaths, p)
	}
	return newPaths, found
}

// endregion

// SystemEnv 返回操作系统的用户级环境变量
func (s *EnvService) SystemEnv() (map[string]string, error) {
	return sysenv.EnvVars()
}

// SystemPaths 返回操作系统的用户级 PATH 条目
func (s *EnvService) SystemPaths() ([]string, error) {
	return sysenv.PathList()
}

// parseEnvMap 解析 KEY=VALUE 列表为 map, 键名统一大写并校验
func parseEnvMap(envs []string) (map[string]string, error) {
	pairs, err := parseEnvPairs(envs)
	if err != nil {
		return nil, err
	}

	envMap := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		envMap[pair.name] = pair.value
	}
	return envMap, nil
}

// normalizeEnvName 规范化并校验环境变量名称
func normalizeEnvName(name string) (string, error) {
	name = strings.ToUpper(name)
	if !strutil.IsVarName(name) {
		return "", fmt.Errorf("invalid environment variable name: %s", name)
	}
	return name, nil
}
