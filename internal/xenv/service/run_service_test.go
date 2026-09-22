package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gookit/goutil/jsonutil"
	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/util"
	"github.com/inhere/xenv/internal/xenv/manager"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

func TestComposeEnvAppliesOrder(t *testing.T) {
	base := []string{"PATH=/bin", "APP_ENV=dev", "KEEP=1"}
	sdkEnvs := map[string]string{"GOROOT": "/go", "APP_ENV": "sdk"}

	env, added, err := composeEnv(base, sdkEnvs, []string{"APP_ENV=cli", "NEW=2"})
	assert.Require(t, assert.NoErr(t, err))

	// SDK active_env 按名称排序, --env 按输入顺序; 被覆盖的变量只输出最终值
	assert.Eq(t, []string{"APP_ENV=cli", "GOROOT=/go", "NEW=2"}, added)
	assert.Eq(t, "cli", envValue(env, "APP_ENV"))
	assert.Eq(t, "/go", envValue(env, "GOROOT"))
	assert.Eq(t, "1", envValue(env, "KEEP"))
	assert.Eq(t, "2", envValue(env, "NEW"))
	// 覆盖时保持原位置, 不产生重复条目
	assert.Eq(t, 5, len(env))

	t.Run("empty value is allowed", func(t *testing.T) {
		env, _, err := composeEnv(nil, nil, []string{"EMPTY="})
		assert.Require(t, assert.NoErr(t, err))
		assert.Eq(t, "EMPTY=", env[0])
	})
}

func TestComposeEnvRejectsInvalidPairs(t *testing.T) {
	t.Run("missing equals", func(t *testing.T) {
		_, _, err := composeEnv(nil, nil, []string{"APP_ENV"})
		assert.ErrMsgContains(t, err, "expected KEY=VALUE")
	})

	t.Run("invalid name", func(t *testing.T) {
		_, _, err := composeEnv(nil, nil, []string{"1BAD=1"})
		assert.ErrMsgContains(t, err, "invalid environment variable name")
	})
}

func TestComposePathOrderAndDedupe(t *testing.T) {
	oldShell := xenvcom.HookShell()
	xenvcom.SetHookShell("")
	t.Cleanup(func() { xenvcom.SetHookShell(oldShell) })

	sep := xenvcom.PathSep()
	basePath := strings.Join([]string{"/bin", "/usr/bin"}, sep)

	// 顺序: --path -> SDK bin -> 继承 PATH; 重复条目保留首次出现的位置
	got := composePath([]string{"/tools", "/bin"}, []string{"/go/bin"}, basePath)
	assert.Eq(t, strings.Join([]string{"/tools", "/bin", "/go/bin", "/usr/bin"}, sep), got)

	// 空条目丢弃
	assert.Eq(t, "", composePath(nil, nil, sep+sep))
	assert.Eq(t, "", composePath(nil, nil, ""))

	t.Run("windows ignores case and separator", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("windows path compare only")
		}
		got := composePath([]string{`C:\Tools\`}, nil, `c:/tools;C:\bin`)
		assert.Eq(t, `C:\Tools\;C:\bin`, got)
	})

	t.Run("unix keeps case", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("unix path compare only")
		}
		assert.Eq(t, "/Bin:/bin", composePath([]string{"/Bin"}, nil, "/bin"))
	})
}

func TestBuildEnvValidatesOptions(t *testing.T) {
	svc := newRunTestService(t, newRunTestConfig(), nil)
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")
	assert.Require(t, assert.NoErr(t, os.WriteFile(filePath, []byte("x"), 0o644)))

	tests := map[string]struct {
		opts   RunOptions
		errMsg string
	}{
		"cwd is a file":        {RunOptions{Cwd: filePath}, "cwd is not a directory"},
		"cwd is missing":       {RunOptions{Cwd: filepath.Join(dir, "nope")}, "cwd is not a directory"},
		"path is missing":      {RunOptions{Paths: []string{filepath.Join(dir, "nope")}}, "path does not exist"},
		"path is a file":       {RunOptions{Paths: []string{filePath}}, "path does not exist"},
		"sdk is not defined":   {RunOptions{Use: []string{"rust:1.0"}}, "config is not defined"},
		"sdk is not installed": {RunOptions{Use: []string{"go:9.9.9"}}, "is not installed locally"},
		"invalid env pair":     {RunOptions{Envs: []string{"APP_ENV"}}, "expected KEY=VALUE"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := svc.BuildEnv(test.opts)
			assert.ErrMsgContains(t, err, test.errMsg)
		})
	}
}

func TestBuildEnvResolvesSDKs(t *testing.T) {
	installDir := filepath.Join(t.TempDir(), "tools", "go", "1.24.0")
	binDir := filepath.Join(installDir, "bin")
	extraDir := t.TempDir()
	workDir := t.TempDir()

	cfg := newRunTestConfig()
	svc := newRunTestService(t, cfg, []models.InstalledSDK{{
		ID:         "go:1.24.0",
		Name:       "go",
		Version:    "1.24.0",
		InstallDir: installDir,
	}})

	runEnv, err := svc.BuildEnv(RunOptions{
		Use:   []string{"go:1.24"},
		Paths: []string{extraDir},
		Envs:  []string{"APP_ENV=local"},
		Cwd:   workDir,
	})
	assert.Require(t, assert.NoErr(t, err))

	assert.Eq(t, util.NormalizePath(workDir), runEnv.Cwd)
	assert.Eq(t, 1, len(runEnv.SDKs))
	assert.Eq(t, "1.24.0", runEnv.SDKs[0].Version)
	assert.True(t, hasEnvItem(runEnv.Envs, "APP_ENV=local"))
	assert.True(t, hasEnvItem(runEnv.Envs, "GOROOT="+util.NormalizePath(installDir)))

	// PATH 顺序: --path -> SDK bin -> 继承 PATH
	sep := xenvcom.PathSep()
	prefix := strings.Join([]string{util.NormalizePath(extraDir), util.NormalizePath(binDir)}, sep) + sep
	assert.True(t, strings.HasPrefix(runEnv.Path, prefix), "unexpected PATH: %s", runEnv.Path)
	assert.Eq(t, runEnv.Path, envValue(runEnv.Env, "PATH"))
	assert.Eq(t, "local", envValue(runEnv.Env, "APP_ENV"))
	assert.Eq(t, util.NormalizePath(installDir), envValue(runEnv.Env, "GOROOT"))
}

func TestRunWithoutCommand(t *testing.T) {
	svc := newRunTestService(t, newRunTestConfig(), nil)

	_, err := svc.Run(&RunEnv{}, nil)
	assert.ErrMsgContains(t, err, "command is required")
}

func TestRunCommandNotFound(t *testing.T) {
	svc := newRunTestService(t, newRunTestConfig(), nil)

	code, err := svc.Run(&RunEnv{Path: t.TempDir()}, []string{"xenv-not-exist-cmd"})
	assert.Eq(t, 127, code)
	assert.ErrMsgContains(t, err, "command not found")
}

// newRunTestConfig 返回只包含 go 的最小配置
func newRunTestConfig() *models.Configuration {
	return &models.Configuration{
		SDKs: []models.ToolChain{{
			Name:      "go",
			BinDir:    "bin",
			ActiveEnv: map[string]string{"GOROOT": "{install_dir}"},
		}},
	}
}

// newRunTestService 使用临时索引文件构造 RunService, 不触碰用户真实索引
func newRunTestService(t *testing.T, cfg *models.Configuration, sdks []models.InstalledSDK) *RunService {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	indexFile := filepath.Join(tempHome, "sdks.local.json")
	assert.Require(t, assert.NoErr(t, jsonutil.WritePretty(indexFile, &models.SDKLocalIndex{
		Schema: 1,
		SDKs:   sdks,
	})))

	sdkMgr := manager.NewSDKManager(indexFile)
	assert.Require(t, assert.NoErr(t, sdkMgr.Init(cfg)))
	return NewRunService(NewSDKService(cfg, manager.NewStateManager(), sdkMgr))
}

// envValue 返回环境变量列表中的值
func envValue(env []string, name string) string {
	for _, item := range env {
		if itemName, value, ok := strings.Cut(item, "="); ok && sameEnvName(itemName, name) {
			return value
		}
	}
	return ""
}

func hasEnvItem(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}
