package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/gookit/goutil/fsutil"
	"github.com/inhere/xenv/internal/util"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/sdk"
	"github.com/inhere/xenv/internal/xenv/sysenv"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

// exitCodeNotFound 命令不存在时的退出码, 与 shell 约定保持一致
const exitCodeNotFound = 127

// RunOptions 一次性执行命令的选项
type RunOptions struct {
	// Use 一次性激活的 SDK 规格列表, eg: go:1.24
	Use []string
	// Paths 追加到 PATH 最前面的目录列表
	Paths []string
	// Envs 环境变量设置项列表, 格式 KEY=VALUE
	Envs []string
	// Cwd 子进程的工作目录, 为空表示继承当前目录
	Cwd string
}

// RunEnv 合成后的执行环境
type RunEnv struct {
	// Env 子进程的完整环境变量
	Env []string
	// Path 合成后的 PATH 值
	Path string
	// Envs 本次由选项新增或覆盖的 KEY=VALUE, 用于 --print
	Envs []string
	// SDKs 解析到的 SDK 列表
	SDKs []*models.InstalledSDK
	// Cwd 子进程的工作目录
	Cwd string
}

// RunService 负责一次性环境的合成与执行
//
// NOTE: 不持有 StateManager, 不写任何 xenv 状态文件
type RunService struct {
	sdkSvc *SDKService
}

// NewRunService creates a RunService
func NewRunService(sdkSvc *SDKService) *RunService {
	return &RunService{sdkSvc: sdkSvc}
}

// ResolveSDKs 解析 SDK 规格为本地安装信息
func (s *RunService) ResolveSDKs(specs []string) ([]*models.InstalledSDK, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	sdkList := make([]*models.InstalledSDK, 0, len(specs))
	for _, item := range specs {
		spec, err := sdk.ParseVersionSpec(item)
		if err != nil {
			return nil, err
		}

		localSDK, err := s.sdkSvc.checkActivateSDK(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve sdk %q: %w", item, err)
		}
		sdkList = append(sdkList, localSDK)
	}
	return sdkList, nil
}

// BuildEnv 合成一次性执行环境
//
// 环境变量顺序: 继承环境 -> SDK active_env -> --env
// PATH 顺序: --path -> SDK bin 目录 -> 继承 PATH
func (s *RunService) BuildEnv(opts RunOptions) (*RunEnv, error) {
	sdkList, err := s.ResolveSDKs(opts.Use)
	if err != nil {
		return nil, err
	}

	cwd, err := resolveRunCwd(opts.Cwd)
	if err != nil {
		return nil, err
	}

	cliPaths, err := checkRunPaths(opts.Paths)
	if err != nil {
		return nil, err
	}

	sdkEnvs := make(map[string]string)
	var sdkBins []string
	for _, localSDK := range sdkList {
		sdkBins = append(sdkBins, localSDK.BinDirPath())
		for name, value := range localSDK.RenderActiveEnv() {
			sdkEnvs[strings.ToUpper(name)] = value
		}
	}

	env, added, err := composeEnv(os.Environ(), sdkEnvs, opts.Envs)
	if err != nil {
		return nil, err
	}

	pathValue := composePath(cliPaths, sdkBins, os.Getenv("PATH"))
	return &RunEnv{
		Env:  setEnvValue(env, "PATH", pathValue),
		Path: pathValue,
		Envs: added,
		SDKs: sdkList,
		Cwd:  cwd,
	}, nil
}

// Run 使用合成环境执行命令, 返回子进程的退出码
func (s *RunService) Run(runEnv *RunEnv, command []string) (int, error) {
	if len(command) == 0 {
		return 0, errors.New("command is required")
	}

	binPath, err := lookPathIn(runEnv.Path, command[0])
	if err != nil {
		return exitCodeNotFound, fmt.Errorf("command not found: %s", command[0])
	}

	cmd := &exec.Cmd{
		Path:   binPath,
		Args:   command,
		Env:    runEnv.Env,
		Dir:    runEnv.Cwd,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err = cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitCodeOf(exitErr), nil
		}
		return 1, fmt.Errorf("failed to run %s: %w", command[0], err)
	}
	return 0, nil
}

// exitCodeOf 返回子进程退出码. 被信号终止时按 shell 约定返回 128+signal
func exitCodeOf(exitErr *exec.ExitError) int {
	if code := exitErr.ExitCode(); code >= 0 {
		return code
	}

	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return 1
}

// lookPathIn 在指定的 PATH 值中查找可执行文件
//
// NOTE: 标准库 exec.LookPath 只读取当前进程的 PATH, 因此临时替换 PATH 查找后立即恢复
func lookPathIn(pathValue, name string) (string, error) {
	oldPath, hasPath := os.LookupEnv("PATH")
	if err := os.Setenv("PATH", pathValue); err != nil {
		return "", err
	}
	defer func() {
		if hasPath {
			_ = os.Setenv("PATH", oldPath)
			return
		}
		_ = os.Unsetenv("PATH")
	}()

	return exec.LookPath(name)
}

// resolveRunCwd 校验子进程的工作目录
func resolveRunCwd(cwd string) (string, error) {
	if cwd == "" {
		return "", nil
	}

	cwd = fsutil.ToAbsPath(util.NormalizePath(cwd))
	info, err := os.Stat(cwd)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("cwd is not a directory: %s", cwd)
	}
	return cwd, nil
}

// checkRunPaths 校验并规范化 --path 目录
//
// 统一转换为绝对路径, 避免子进程工作目录不同导致相对路径失效
func checkRunPaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	cliPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		item := fsutil.ToAbsPath(util.NormalizePath(path))
		info, err := os.Stat(item)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("path does not exist: %s", item)
		}
		cliPaths = append(cliPaths, item)
	}
	return cliPaths, nil
}

// envPair 环境变量键值对
type envPair struct {
	name  string
	value string
}

// parseEnvPairs 解析 KEY=VALUE 列表, 键名统一大写并校验
func parseEnvPairs(envs []string) ([]envPair, error) {
	pairs := make([]envPair, 0, len(envs))
	for _, item := range envs {
		name, value, ok := strings.Cut(item, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid environment pair %q, expected KEY=VALUE", item)
		}

		name, err := normalizeEnvName(name)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, envPair{name: name, value: value})
	}
	return pairs, nil
}

// composeEnv 合成环境变量列表: base -> sdkEnvs -> cliEnvs, 后者覆盖前者
//
// return 合成后的环境变量, 以及本次新增或覆盖的项(用于 --print)
func composeEnv(base []string, sdkEnvs map[string]string, cliEnvs []string) ([]string, []string, error) {
	cliPairs, err := parseEnvPairs(cliEnvs)
	if err != nil {
		return nil, nil, err
	}

	env := make([]string, 0, len(base)+len(sdkEnvs)+len(cliPairs))
	env = append(env, base...)

	// 记录本次涉及的变量名(保持首次出现顺序), 最终按合成后的值输出
	var appliedNames []string
	markApplied := func(name string) {
		for _, item := range appliedNames {
			if sameEnvName(item, name) {
				return
			}
		}
		appliedNames = append(appliedNames, name)
	}

	// SDK 的 active_env 按名称排序, 保证输出稳定
	for _, name := range sortedNames(sdkEnvs) {
		env = setEnvValue(env, name, sdkEnvs[name])
		markApplied(name)
	}

	for _, pair := range cliPairs {
		env = setEnvValue(env, pair.name, pair.value)
		markApplied(pair.name)
	}

	added := make([]string, 0, len(appliedNames))
	for _, name := range appliedNames {
		added = append(added, name+"="+envValueOf(env, name))
	}
	return env, added, nil
}

// envValueOf 返回环境变量列表中的值
func envValueOf(env []string, name string) string {
	for _, item := range env {
		itemName, value, ok := strings.Cut(item, "=")
		if ok && sameEnvName(itemName, name) {
			return value
		}
	}
	return ""
}

// composePath 合成 PATH 值: --path 条目 -> SDK bin 目录 -> 继承 PATH
//
// 重复条目只保留首次出现的位置, 空条目丢弃
func composePath(cliPaths, sdkBins []string, basePath string) string {
	items := make([]string, 0, len(cliPaths)+len(sdkBins)+4)
	items = append(items, cliPaths...)
	items = append(items, sdkBins...)
	items = append(items, util.SplitPath(basePath)...)

	seen := make(map[string]struct{}, len(items))
	pathList := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item == "" {
			continue
		}

		key := pathCompareKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		pathList = append(pathList, item)
	}
	return strings.Join(pathList, xenvcom.PathSep())
}

// pathCompareKey 生成 PATH 条目比较用的 key
func pathCompareKey(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(sysenv.NormalizeWinPath(path))
	}
	return strings.TrimRight(path, "/")
}

// setEnvValue 设置环境变量列表中的值, 变量不存在时追加
func setEnvValue(env []string, name, value string) []string {
	for i, item := range env {
		itemName, _, ok := strings.Cut(item, "=")
		if !ok || !sameEnvName(itemName, name) {
			continue
		}

		env[i] = name + "=" + value
		return env
	}
	return append(env, name+"="+value)
}

// sameEnvName 比较环境变量名称. Windows 下忽略大小写
func sameEnvName(name1, name2 string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(name1, name2)
	}
	return name1 == name2
}

// sortedNames 返回排序后的 map 键
func sortedNames(values map[string]string) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
