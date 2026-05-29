# xenv SDK Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `xenv` 收缩为本地 SDK 发现、索引、激活和项目环境检查工具，删除下载/安装职责，统一用户级状态目录到 `~/.config/xenv`，并实现可选 `eget` SDK store 读取。

**Architecture:** 以 `sdk` 为唯一 SDK 命名空间，删除 `tools` 命令和下载实现。新增 `SDKManager`/`SDKService` 负责 SDK 索引、查找和激活；配置目录由 `XENV_CONFIG_DIR` 或默认 `~/.config/xenv` 派生；`.xenv.toml [tools]` 作为项目外部 CLI 工具需求，由 `xenv check tools` 检查。

**Tech Stack:** Go, `gookit/gcli/v3`, TOML/JSON, PowerShell/bash/zsh hook script generation.

---

## 复审结论

设计可以实施，没有阻断性冲突。需要在实现中重点处理以下风险：

- 当前 `internal/cli/app.go` 仍注册 `ToolsCmd`，需要替换为 `SDKCmd`。
- 当前 `internal/cli/list_cmd.go` 里还有 `list tools` 和 `tools` alias，必须改成 `list sdk`。
- 当前 `ActivityState.Tools` 实际会参与 TOML merge/save，需要改名为 `ToolRequirements`，但 TOML/JSON 字段仍保持 `tools`。
- 当前 `StateTomlUpdater` 明确处理 `[tools]`，需要同步使用 `ToolRequirements`。
- 当前 zsh hook `chpwd` 调用 `init-direnv` 后没有 eval 返回表达式；实现项目脚本 source 时要一起补 hook 测试。
- `~/.xenv` 到 `~/.config/xenv` 是不兼容迁移，项目仍处本地开发阶段，可以直接切换，不做兼容读取。

## 文件结构

计划创建或修改这些文件：

- `internal/xenv/config/paths.go`: 新增配置目录路径解析，支持 `XENV_CONFIG_DIR`。
- `internal/xenv/config/config.go`: 删除下载默认配置，新增 `eget_enable`、`eget_store_file`、`check_tools_on_direnv`、`source_project_scripts` 默认值。
- `internal/xenv/models/config.go`: 删除下载字段和 simple tools 字段，新增配置字段。
- `internal/xenv/models/state.go`: `Tools` 改为 `ToolRequirements`，TOML/JSON tag 仍为 `tools`。
- `internal/xenv/models/sdk_local.go`: 新增 SDK 专用本地索引结构。
- `internal/xenv/manager/sdk_manager.go`: 替代 `tool_manager.go`，负责 xenv 本地索引、eget store 读取、合并和版本匹配。
- `internal/xenv/manager/state_manager.go`: 使用新的配置目录和 `ToolRequirements`。
- `internal/xenv/manager/state_toml_update.go`: `[tools]` 写入改为使用 `ToolRequirements`。
- `internal/xenv/sdk/version.go`: 从 `internal/xenv/tools/version.go` 移入 SDK 包。
- `internal/xenv/service/sdk_service.go`: 替代 `tool_service.go` 中的 SDK 激活、索引、列表、where 逻辑，删除 install/update/uninstall/register。
- `internal/xenv/service/check_service.go`: 新增 `.xenv.toml [tools]` 和 SDK/path 检查服务。
- `internal/xenv/shell/gen_hook_bash.go`: 支持 `source_project_scripts` 下 source `.xenv.sh`。
- `internal/xenv/shell/gen_hook_zsh.go`: 修复 `init-direnv` eval，并支持 source `.xenv.sh`。
- `internal/xenv/shell/gen_hook_pwsh.go`: 支持 dot-source `.xenv.ps1`。
- `internal/xenv/models/dto.go`: `GenInitScriptParams` 增加项目脚本 source 参数。
- `internal/cli/sdk_cmd.go`: 新增 `sdk index/refresh/scan/list/show/where/which`。
- `internal/cli/check_cmd.go`: 新增 `check/check sdk/check tools`。
- `internal/cli/app.go`: 注册 `SDKCmd` 和 `CheckCmd`，移除 `ToolsCmd`。
- `internal/cli/list_cmd.go`: `list tools` 改为 `list sdk`，清理 tools 文案。
- `internal/cli/use_cmd.go`: 使用 `xenv.SDKService()` 和 `sdk.ParseVersionSpec`。
- `README.md`, `config/config.yaml`, `docs/xenv-feature-report.md`: 更新用户文档和示例。
- 删除：`internal/cli/tools_cmd.go`、`internal/xenv/tools/installer.go`、`internal/xenv/tools/uninstaller.go`。

## Task 1: 配置目录与配置模型

**Files:**
- Create: `internal/xenv/config/paths_test.go`
- Create: `internal/xenv/config/paths.go`
- Modify: `internal/xenv/config/config.go`
- Modify: `internal/xenv/models/config.go`
- Test: `go test ./internal/xenv/config`

- [x] Step 1: 写配置目录测试

在 `internal/xenv/config/paths_test.go` 中增加测试：

```go
package config

import (
	"path/filepath"
	"testing"
)

func TestResolveDirUsesDefaultConfigDir(t *testing.T) {
	t.Setenv("XENV_CONFIG_DIR", "")
	home := t.TempDir()

	dir := ResolveDir(func() (string, error) { return home, nil })

	want := filepath.Join(home, ".config", "xenv")
	if dir != want {
		t.Fatalf("ResolveDir() = %q, want %q", dir, want)
	}
}

func TestResolveDirUsesXenvConfigDir(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "xenv-dev")
	t.Setenv("XENV_CONFIG_DIR", custom)

	dir := ResolveDir(func() (string, error) { return "ignored", nil })

	if dir != custom {
		t.Fatalf("ResolveDir() = %q, want %q", dir, custom)
	}
}

func TestDerivedPathsUseConfigDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "xenv")
	paths := PathsForDir(dir)

	checks := map[string]string{
		"config": paths.ConfigFile,
		"global": paths.GlobalStateFile,
		"session": paths.SessionDir,
		"index": paths.SDKLocalIndexFile,
		"hooks": paths.ShellHooksDir,
	}
	wants := map[string]string{
		"config": filepath.Join(dir, "config.yaml"),
		"global": filepath.Join(dir, "global.toml"),
		"session": filepath.Join(dir, "session"),
		"index": filepath.Join(dir, "sdks.local.json"),
		"hooks": filepath.Join(dir, "hooks"),
	}
	for name, got := range checks {
		if got != wants[name] {
			t.Fatalf("%s path = %q, want %q", name, got, wants[name])
		}
	}
}
```

- [x] Step 2: 运行测试确认失败

Run:

```bash
go test ./internal/xenv/config -run 'TestResolveDir|TestDerivedPaths' -count=1
```

Expected: fail，提示 `ResolveDir`、`PathsForDir` 未定义。

- [x] Step 3: 实现配置目录解析

在 `internal/xenv/config/paths.go` 中实现：

```go
package config

import (
	"os"
	"path/filepath"
)

const EnvConfigDir = "XENV_CONFIG_DIR"

type Paths struct {
	ConfigDir         string
	ConfigFile        string
	GlobalStateFile   string
	SessionDir        string
	SDKLocalIndexFile string
	ShellHooksDir     string
}

func ResolveDir(homeFn func() (string, error)) string {
	if dir := os.Getenv(EnvConfigDir); dir != "" {
		return filepath.Clean(dir)
	}
	home, err := homeFn()
	if err != nil || home == "" {
		return filepath.Join(".config", "xenv")
	}
	return filepath.Join(home, ".config", "xenv")
}

func PathsForDir(dir string) Paths {
	return Paths{
		ConfigDir:         dir,
		ConfigFile:        filepath.Join(dir, "config.yaml"),
		GlobalStateFile:   filepath.Join(dir, "global.toml"),
		SessionDir:        filepath.Join(dir, "session"),
		SDKLocalIndexFile: filepath.Join(dir, "sdks.local.json"),
		ShellHooksDir:     filepath.Join(dir, "hooks"),
	}
}

func DefaultPaths() Paths {
	return PathsForDir(ResolveDir(os.UserHomeDir))
}
```

- [x] Step 4: 修改配置模型

在 `internal/xenv/models/config.go` 中：

```go
type Configuration struct {
	BinDir               string            `json:"bin_dir"`
	EgetEnable           bool              `json:"eget_enable"`
	EgetStoreFile        string            `json:"eget_store_file"`
	CheckToolsOnDirenv   bool              `json:"check_tools_on_direnv"`
	SourceProjectScripts bool              `json:"source_project_scripts"`
	ShellAliases         map[string]string `json:"shell_aliases"`
	ShellHooksDir        string            `json:"shell_hooks_dir"`
	GlobalEnv            map[string]string `json:"global_env"`
	GlobalPaths          []string          `json:"global_paths"`
	AllowUpMatch         uint8             `json:"allow_up_match"`
	SDKs                 []ToolChain       `json:"sdks"`
	configFile           string
	configDir            string
}
```

从 `ToolChain` 删除：

```go
InstallURL string
DownloadExt map[string]string
PostInstall []string
```

保留：

```go
Name string
Alias string
InstallDir string
ActiveEnv map[string]string
BinDir string
OtherVersions map[string]string
```

- [x] Step 5: 更新默认配置加载

在 `internal/xenv/config/config.go`：

- 删除 `DefaultInstallDir`。
- `DefaultShellHooksDir` 改为从 `DefaultPaths().ShellHooksDir` 读取。
- `NewConfigManager()` 默认设置新增字段。
- `GetDefaultConfigPath()` 返回 `DefaultPaths().ConfigFile`。
- `GetDefaultConfigDir()` 返回 `DefaultPaths().ConfigDir`。

- [x] Step 6: 运行测试

Run:

```bash
go test ./internal/xenv/config -count=1
```

Expected: pass。

- [x] Step 7: 提交

```bash
git add internal/xenv/config internal/xenv/models/config.go
git commit -m "refactor: centralize xenv config paths"
```

## Task 2: SDK 本地索引和 manager 重命名

**Files:**
- Create: `internal/xenv/models/sdk_local.go`
- Create: `internal/xenv/manager/sdk_manager_test.go`
- Create: `internal/xenv/manager/sdk_manager.go`
- Delete: `internal/xenv/models/tool_local.go`
- Delete: `internal/xenv/manager/tool_manager.go`
- Modify: `internal/xenv/xenvcom/const.go`
- Test: `go test ./internal/xenv/manager ./internal/xenv/models`

- [x] Step 1: 写 SDK 本地索引测试

在 `internal/xenv/manager/sdk_manager_test.go` 中增加：

```go
package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inhere/xenv/internal/xenv/models"
)

func TestSDKManagerIndexLocalSDKsWritesSDKOnlyIndex(t *testing.T) {
	root := t.TempDir()
	go122 := filepath.Join(root, "go1.22.0")
	go123 := filepath.Join(root, "go1.23.1")
	if err := os.MkdirAll(filepath.Join(go122, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(go123, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	indexFile := filepath.Join(t.TempDir(), "sdks.local.json")
	mgr := NewSDKManager(indexFile)
	err := mgr.Init(&models.Configuration{
		SDKs: []models.ToolChain{{
			Name: "go",
			InstallDir: filepath.Join(root, "go{version}"),
			BinDir: "bin",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.IndexLocalSDKs(); err != nil {
		t.Fatal(err)
	}

	idx, err := mgr.LoadLocalIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.SDKs) != 2 {
		t.Fatalf("indexed SDKs = %d, want 2", len(idx.SDKs))
	}
	if len(idx.Tools) != 0 {
		t.Fatalf("index must not expose tools")
	}
}
```

如果 `SDKLocalIndex` 不包含 `Tools` 字段，则删除测试中 `idx.Tools` 检查，改为读取 JSON 字符串确认不包含 `"tools"`：

```go
data, _ := os.ReadFile(indexFile)
if strings.Contains(string(data), `"tools"`) { t.Fatal("unexpected tools field") }
```

- [x] Step 2: 运行测试确认失败

Run:

```bash
go test ./internal/xenv/manager -run TestSDKManagerIndexLocalSDKsWritesSDKOnlyIndex -count=1
```

Expected: fail，`NewSDKManager` 或 `SDKLocalIndex` 未定义。

- [x] Step 3: 新增模型

在 `internal/xenv/models/sdk_local.go`：

```go
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
		"name": s.Name,
		"version": s.Version,
		"install_dir": util.NormalizePath(s.InstallDir),
	})
}
```

- [x] Step 4: 实现 SDKManager

将 `internal/xenv/manager/tool_manager.go` 迁移为 `sdk_manager.go`：

- 类型名：`SDKManager`
- 构造函数：`NewSDKManager(indexFile string) *SDKManager`
- 方法：
  - `Init(config *models.Configuration) error`
  - `LoadLocalIndex() (*models.SDKLocalIndex, error)`
  - `IndexLocalSDKs() error`
  - `ListSDKVersions(name string) []models.InstalledSDK`
  - `MatchSDKByVersion(local []models.InstalledSDK, version string) *models.InstalledSDK`
  - `MatchSDKByNameAndVersion(name, version string) *models.InstalledSDK`

索引文件路径由构造函数注入，生产代码传 `config.DefaultPaths().SDKLocalIndexFile`。

- [x] Step 5: 更新引用

全仓替换：

- `ToolManager` -> `SDKManager`
- `NewToolManager` -> `NewSDKManager`
- `ToolMgr` -> `SDKMgr`
- `InstalledTool` -> `InstalledSDK`
- `ToolsLocal` -> `SDKLocalIndex`
- `IndexLocalTools` -> `IndexLocalSDKs`

- [x] Step 6: 删除旧模型和 manager

删除：

```text
internal/xenv/models/tool_local.go
internal/xenv/manager/tool_manager.go
```

- [x] Step 7: 运行测试

Run:

```bash
go test ./internal/xenv/manager ./internal/xenv/models -count=1
```

Expected: pass。

- [x] Step 8: 提交

```bash
git add internal/xenv/models internal/xenv/manager internal/xenv/xenvcom/const.go
git commit -m "refactor: replace tool index with sdk index"
```

## Task 3: SDKService 和版本包迁移

**Files:**
- Create: `internal/xenv/sdk/version.go`
- Create: `internal/xenv/service/sdk_service.go`
- Modify: `internal/xenv/xenv.go`
- Modify: `internal/xenv/service/tool_service.go` 或删除后替换
- Delete: `internal/xenv/tools/version.go`
- Delete: `internal/xenv/tools/installer.go`
- Delete: `internal/xenv/tools/uninstaller.go`
- Test: `go test ./internal/xenv/service ./internal/xenv/sdk`

- [x] Step 1: 先移动版本测试

将 `internal/xenv/tools/version_test.go` 移到 `internal/xenv/sdk/version_test.go`，包名改为 `sdk`。

- [x] Step 2: 运行测试确认失败

Run:

```bash
go test ./internal/xenv/sdk -count=1
```

Expected: fail，`ParseVersionSpec` 未定义。

- [x] Step 3: 移动版本实现

将 `internal/xenv/tools/version.go` 移到 `internal/xenv/sdk/version.go`，包名改为 `sdk`。

- [x] Step 4: 新建 SDKService

从 `ToolService` 提取保留能力到 `internal/xenv/service/sdk_service.go`：

```go
type SDKService struct {
	config *models.Configuration
	state  *manager.StateManager
	sdks   *manager.SDKManager
}
```

保留或新增方法：

- `IndexLocalSDKs() error`
- `ListSDKs(showAll bool) error`
- `ShowSDK(name string) error`
- `WhereSDK(spec string, bin bool) (string, error)`
- `ActivateSDKs(useSDKs []string, opFlag models.OpFlag) (string, error)`
- `DeactivateSDKs(deSDKs []string, opFlag models.OpFlag) (string, error)`
- `SetupDirenv() (string, error)`
- `WriteHookToProfile(st shell.ShType, pwshProfile string) error`
- `GenHookScripts(st shell.ShType) (string, error)`

删除方法：

- `Register`
- `InstallTool`
- `UpdateTool`
- `Uninstall`

- [x] Step 5: 更新 xenv facade

在 `internal/xenv/xenv.go`：

- `sdkMgr := manager.NewSDKManager(config.DefaultPaths().SDKLocalIndexFile)`
- `func SDKMgr() *manager.SDKManager`
- `func SDKService() (*service.SDKService, error)`
- 删除 `ToolMgr()` 和 `ToolService()`。

- [x] Step 6: 更新导入

全仓把 `internal/xenv/tools` 的版本解析引用改成 `internal/xenv/sdk`。

- [x] Step 7: 删除下载实现

删除：

```text
internal/xenv/tools/installer.go
internal/xenv/tools/uninstaller.go
```

如果 `internal/xenv/tools` 目录空了，删除目录。

- [x] Step 8: 运行测试

Run:

```bash
go test ./internal/xenv/sdk ./internal/xenv/service -count=1
```

Expected: pass。

- [x] Step 9: 提交

```bash
git add internal/xenv
git commit -m "refactor: rename tool service to sdk service"
```

## Task 4: CLI sdk 命令替换 tools 命令

**Files:**
- Create: `internal/cli/sdk_cmd.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`
- Modify: `internal/cli/list_cmd.go`
- Modify: `internal/cli/use_cmd.go`
- Delete: `internal/cli/tools_cmd.go`
- Test: `go test ./internal/cli`

- [x] Step 1: 更新 CLI 注册测试

修改 `internal/cli/app_test.go`：

```go
func TestNewAppRegistersTopLevelCommands(t *testing.T) {
	app := NewApp()

	if app.Name != "xenv" {
		t.Fatalf("expected app name xenv, got %q", app.Name)
	}

	for _, name := range []string{
		"sdk",
		"use",
		"unuse",
		"env",
		"path",
		"config",
		"list",
		"init",
		"shell",
		"shell-init-hook",
		"shell-direnv",
	} {
		if !app.HasCommand(name) {
			t.Fatalf("expected app to register command %q", name)
		}
	}

	if app.HasCommand("tools") {
		t.Fatalf("tools command must not be registered")
	}
}
```

- [x] Step 2: 运行测试确认失败

Run:

```bash
go test ./internal/cli -run TestNewAppRegistersTopLevelCommands -count=1
```

Expected: fail，`sdk` 未注册或 `tools` 仍存在。

- [x] Step 3: 新增 sdk 命令

在 `internal/cli/sdk_cmd.go` 实现：

- `SDKCmd`
- `SDKIndexCmd`，`Aliases: []string{"refresh", "scan"}`
- `SDKListCmd`
- `SDKShowCmd`
- `SDKWhereCmd`，`Aliases: []string{"which"}`

所有实现调用 `xenv.SDKService()`。

- [x] Step 4: 更新 app 注册

在 `internal/cli/app.go`：

- 删除 `ToolsCmd`
- 新增 `SDKCmd`
- Desc 改为 `Manage local development environments and SDK activation`。

- [x] Step 5: 更新 list 命令

在 `internal/cli/list_cmd.go`：

- `ListToolsCmd` 改为 `ListSDKCmd`
- 子命令名 `sdk`
- alias 可保留 `sdks`，但不能是 `tools`/`tool`
- 默认 `xenv list` 调用 `listSDKs()`
- 文案从 tools 改为 SDKs

- [x] Step 6: 更新 use 命令

在 `internal/cli/use_cmd.go`：

- 参数名从 `tools` 改为 `sdks`
- 变量从 `toolSvc` 改为 `sdkSvc`
- 调用 `xenv.SDKService()`

- [x] Step 7: 删除旧 tools 命令

删除 `internal/cli/tools_cmd.go`。

- [x] Step 8: 运行测试

Run:

```bash
go test ./internal/cli -count=1
```

Expected: pass。

- [x] Step 9: 提交

```bash
git add internal/cli
git commit -m "refactor: replace tools command with sdk command"
```

## Task 5: eget store source 与合并列表

**Files:**
- Create: `internal/xenv/manager/eget_store_test.go`
- Create: `internal/xenv/manager/eget_store.go`
- Modify: `internal/xenv/manager/sdk_manager.go`
- Test: `go test ./internal/xenv/manager`

- [x] Step 1: 写 eget store 映射测试

在 `internal/xenv/manager/eget_store_test.go`：

```go
package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEgetStoreSourceListsSDKs(t *testing.T) {
	store := filepath.Join(t.TempDir(), "sdk.installed.json")
	data := []byte(`{
	  "schema": 1,
	  "installed": {
	    "go": {
	      "versions": {
	        "1.22.0": {
	          "name": "go",
	          "version": "1.22.0",
	          "path": "D:/sdk/go1.22.0"
	        }
	      }
	    }
	  }
	}`)
	if err := os.WriteFile(store, data, 0o644); err != nil {
		t.Fatal(err)
	}

	src := EgetStoreSource{Path: store}
	items, err := src.ListSDKVersions("go")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Source != "eget" {
		t.Fatalf("source = %q, want eget", items[0].Source)
	}
	if items[0].InstallDir != "D:/sdk/go1.22.0" {
		t.Fatalf("install dir = %q", items[0].InstallDir)
	}
}
```

- [x] Step 2: 运行测试确认失败

Run:

```bash
go test ./internal/xenv/manager -run TestEgetStoreSourceListsSDKs -count=1
```

Expected: fail，`EgetStoreSource` 未定义。

- [x] Step 3: 实现 eget store source

在 `internal/xenv/manager/eget_store.go` 实现读取结构：

```go
type EgetStoreSource struct { Path string }
```

JSON 映射结构：

```go
type egetInstalledStore struct {
	Schema int `json:"schema"`
	Installed map[string]struct {
		Versions map[string]struct {
			Name string `json:"name"`
			Version string `json:"version"`
			Path string `json:"path"`
		} `json:"versions"`
	} `json:"installed"`
}
```

返回 `[]models.InstalledSDK`，`Source: "eget"`。

- [x] Step 4: 实现合并来源

在 `SDKManager` 增加：

- `SetEgetSource(source EgetStoreSource)`
- `ListMergedSDKVersions(name string) []models.InstalledSDK`

规则：

- `config.EgetEnable == false`: 只返回 xenv index。
- `config.EgetEnable == true`: 先读 eget，再读 xenv。
- 同名同版本重复时保留 eget。
- eget store 缺失或损坏时返回 xenv index，并记录 warning。

- [x] Step 5: 运行测试

Run:

```bash
go test ./internal/xenv/manager -count=1
```

Expected: pass。

- [x] Step 6: 提交

```bash
git add internal/xenv/manager
git commit -m "feat: read optional eget sdk store"
```

## Task 6: check 命令和 tools requirements

**Files:**
- Create: `internal/xenv/service/check_service_test.go`
- Create: `internal/xenv/service/check_service.go`
- Create: `internal/cli/check_cmd.go`
- Modify: `internal/xenv/models/state.go`
- Modify: `internal/xenv/manager/state_toml_update.go`
- Modify: `internal/cli/list_cmd.go`
- Test: `go test ./internal/xenv/service ./internal/cli`

- [ ] Step 1: 写 `.xenv.toml [tools]` 检查测试

在 `internal/xenv/service/check_service_test.go`：

```go
package service

import (
	"testing"

	"github.com/inhere/xenv/internal/xenv/models"
)

func TestParseToolRequirementSimpleMap(t *testing.T) {
	tests := []struct {
		raw string
		wantVersion string
		wantRequired bool
	}{
		{"*", "", true},
		{">=1.32", "1.32", true},
		{">=1.32,required", "1.32", true},
		{">=1.32,optional", "1.32", false},
	}
	for _, tt := range tests {
		got, err := ParseToolRequirement(tt.raw)
		if err != nil {
			t.Fatalf("ParseToolRequirement(%q): %v", tt.raw, err)
		}
		if got.MinVersion != tt.wantVersion || got.Required != tt.wantRequired {
			t.Fatalf("ParseToolRequirement(%q) = %+v", tt.raw, got)
		}
	}
}

func TestActivityStateUsesToolRequirementsField(t *testing.T) {
	state := models.NewActivityState(".xenv.toml")
	state.ToolRequirements["rg"] = "*"
	if state.IsEmpty() {
		t.Fatal("state with tool requirements must not be empty")
	}
}
```

- [ ] Step 2: 运行测试确认失败

Run:

```bash
go test ./internal/xenv/service -run 'TestParseToolRequirement|TestActivityStateUsesToolRequirementsField' -count=1
```

Expected: fail，`ParseToolRequirement` 或 `ToolRequirements` 未定义。

- [ ] Step 3: 改 ActivityState

在 `internal/xenv/models/state.go`：

- `Tools` 改为 `ToolRequirements map[string]string`
- tag 保持 `json:"tools" toml:"tools"`
- `AddTools` 改为 `AddToolRequirements`
- `DelTool` 改为 `DelToolRequirement`
- `Merge` 和 `IsEmpty` 使用 `ToolRequirements`

- [ ] Step 4: 更新 TOML updater

在 `internal/xenv/manager/state_toml_update.go`：

- `[tools]` 分支使用 `state.ToolRequirements`
- `sections := []string{"envs", "sdks", "tools"}` 保留

- [ ] Step 5: 实现 CheckService

在 `internal/xenv/service/check_service.go`：

- `ParseToolRequirement(raw string) (ToolRequirement, error)`
- `CheckTools(state *models.ActivityState) []CheckResult`
- 第一版 `CheckTools` 使用 `exec.LookPath`，版本检查只在 `xenv check tools` 时执行 `<tool> --version`。
- 版本解析失败返回 warning。

- [ ] Step 6: 新增 CLI check 命令

在 `internal/cli/check_cmd.go`：

- `CheckCmd`
- `CheckSDKCmd`
- `CheckToolsCmd`

`CheckToolsCmd` 调用 `xenv.InitState()` 后检查 `xenv.State().Merged().ToolRequirements`。

同时在 `internal/cli/app.go` 注册 `CheckCmd`，并把 `internal/cli/app_test.go` 的顶层命令列表加入 `"check"`。

- [ ] Step 7: 运行测试

Run:

```bash
go test ./internal/xenv/service ./internal/cli -count=1
```

Expected: pass。

- [ ] Step 8: 提交

```bash
git add internal/xenv/models internal/xenv/manager internal/xenv/service internal/cli
git commit -m "feat: add xenv check for project tool requirements"
```

## Task 7: shell hook 项目脚本 source

**Files:**
- Modify: `internal/xenv/models/dto.go`
- Modify: `internal/xenv/service/sdk_service.go`
- Modify: `internal/xenv/shell/gen_hook_bash.go`
- Modify: `internal/xenv/shell/gen_hook_zsh.go`
- Modify: `internal/xenv/shell/gen_hook_pwsh.go`
- Modify: `internal/xenv/shell/gen_hook_test.go`
- Test: `go test ./internal/xenv/shell ./internal/xenv/service`

- [ ] Step 1: 写 hook 生成测试

在 `internal/xenv/shell/gen_hook_test.go` 增加：

```go
func TestGeneratedHooksSupportProjectScripts(t *testing.T) {
	params := &models.GenInitScriptParams{
		ShellHooksDir: "~/.config/xenv/hooks",
		SourceProjectScripts: true,
	}

	bash, err := NewScriptGenerator(Bash).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, bash, ".xenv.sh")

	zsh, err := NewScriptGenerator(Zsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, zsh, ".xenv.sh")
	assertContains(t, zsh, "invoke_xenv_result")

	pwsh, err := NewScriptGenerator(Pwsh).GenHookScripts(params)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, pwsh, ".xenv.ps1")
}
```

- [ ] Step 2: 运行测试确认失败

Run:

```bash
go test ./internal/xenv/shell -run TestGeneratedHooksSupportProjectScripts -count=1
```

Expected: fail，模板不包含项目脚本逻辑。

- [ ] Step 3: 扩展 GenInitScriptParams 和脚本生成器

在 `internal/xenv/models/dto.go` 增加：

```go
SourceProjectScripts bool
```

在 `internal/xenv/shell/gen_hook.go` 增加方法：

```go
func (sg *XenvScriptGenerator) GenSourceProjectScript(projectDir string) string
```

行为：

- Bash/Zsh 返回 `source "<projectDir>/.xenv.sh"`，仅当文件存在时由 Go 端决定是否生成。
- PowerShell 返回 `. "<projectDir>/.xenv.ps1"`，仅当文件存在时由 Go 端决定是否生成。
- CMD 第一版不支持项目脚本，返回空字符串。

- [ ] Step 4: 由 `init-direnv` 返回项目脚本 source 表达式

在 `SDKService.SetupDirenv()` 中：

- 找到最近 `.xenv.toml` 对应的 dir state。
- 先生成 SDK/env/path 激活脚本。
- 如果 `config.SourceProjectScripts == true` 且最近 `.xenv.toml` 存在：
  - bash/zsh 检查同目录 `.xenv.sh` 是否存在，存在则 append `gen.GenSourceProjectScript(projectDir)`。
  - pwsh 检查同目录 `.xenv.ps1` 是否存在，存在则 append `gen.GenSourceProjectScript(projectDir)`。
- 如果没有 `.xenv.toml`，不生成项目脚本 source。

这样 shell hook 不需要自己猜项目目录，所有项目脚本逻辑由 `xenv init-direnv` 根据已加载的 dir state 生成。

- [ ] Step 5: Bash/Zsh/Pwsh hook 只负责 eval `init-direnv` 结果

不在 hook 模板中直接查找 `.xenv.sh` 或 `.xenv.ps1`。

修复 zsh `chpwd`：

```zsh
chpwd() {
    if (( $+commands[{{BinName}}] )); then
        local result="$(command {{BinCommand}} init-direnv)"
        local exit_code=$?
        invoke_xenv_result "$result" $exit_code
    fi
}
```

- [ ] Step 6: SDKService 传参

`GenHookScripts` 不需要把 `SourceProjectScripts` 写进 hook 模板，但 `SetupDirenv` 需要使用 `ts.config.SourceProjectScripts` 决定是否 append 项目脚本 source 表达式。

- [ ] Step 7: 运行测试

Run:

```bash
go test ./internal/xenv/shell ./internal/xenv/service -count=1
```

Expected: pass。

- [ ] Step 8: 提交

```bash
git add internal/xenv/models internal/xenv/service internal/xenv/shell
git commit -m "feat: support optional project xenv scripts"
```

## Task 8: 文档、示例和全量验证

**Files:**
- Modify: `README.md`
- Modify: `config/config.yaml`
- Modify: `docs/xenv-feature-report.md`
- Modify: `docs/design/2026-05-29-xenv-sdk-eget-command-design.md` if implementation differs
- Test: `go test ./...`

- [ ] Step 1: 更新 README

更新命令示例：

```bash
xenv sdk index
xenv sdk refresh
xenv sdk scan
xenv sdk list
xenv sdk where go:1.22
xenv sdk which go:1.22
xenv check tools
```

删除：

```bash
xenv tools list
xenv tools index
```

路径改为：

```text
~/.config/xenv/config.yaml
~/.config/xenv/session/<session_id>.json
~/.config/xenv/sdks.local.json
```

- [ ] Step 2: 更新 `config/config.yaml`

保留示例：

```yaml
eget_enable: false
eget_store_file: ""
check_tools_on_direnv: false
source_project_scripts: false
allow_up_match: 1
shell_hooks_dir: "~/.config/xenv/hooks"
global_env: {}
global_paths: []
sdks:
  - name: go
    alias: golang
    install_dir: "D:/work/env/devsdk/gosdk/go{version}"
    bin_dir: "bin"
    active_env:
      GOROOT: "{install_dir}"
```

删除下载字段和 `tools` 配置。

- [ ] Step 3: 更新 feature report

把 CLI 路径、命令结构、数据文件路径改到 `sdk` 和 `~/.config/xenv/sdks.local.json`。

- [ ] Step 4: 全量测试

Run:

```bash
go test ./...
```

Expected: pass。

- [ ] Step 5: CLI smoke test

Run:

```bash
go run ./cmd/xenv sdk --help
go run ./cmd/xenv tools --help
go run ./cmd/xenv sdk index
go run ./cmd/xenv sdk refresh
go run ./cmd/xenv sdk scan
go run ./cmd/xenv sdk list
go run ./cmd/xenv check tools
```

Expected:

- `sdk --help`: shows `index/refresh/scan/list/show/where/which`
- `tools --help`: unknown command
- `go test ./...`: pass

- [ ] Step 6: 提交

```bash
git add README.md config/config.yaml docs
git commit -m "docs: update xenv sdk workflow"
```

## Self-Review

- Spec coverage: 覆盖了 `XENV_CONFIG_DIR`、`sdks.local.json`、`sdk` 命令、`where/which`、`index/refresh/scan`、`eget_enable/eget_store_file`、`check tools`、`source_project_scripts`、删除下载能力和一次性重命名。
- Placeholder scan: 本计划没有使用 TBD/TODO/稍后实现作为任务内容；每个任务都有明确文件、测试和命令。
- Type consistency: 统一使用 `SDKManager`、`SDKService`、`SDKLocalIndex`、`InstalledSDK`、`ToolRequirements`、`SDKCmd` 命名。
