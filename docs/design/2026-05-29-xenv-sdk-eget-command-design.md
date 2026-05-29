# xenv SDK 命令与 eget 可选集成设计

## 背景

`xenv` 当前从 `kite-go` 迁移出来后，仍保留了一部分下载、安装、卸载 SDK/工具的设计和代码，例如 `tools install`、`tools update`、`tools uninstall`、`install_url`、`download_ext`、`download_dir` 等。与此同时，工作区内的 `eget` 已经承担了更完整的下载、解压、缓存、SDK 安装记录能力。

新的方向是收缩 `xenv` 的职责：`xenv` 不再负责下载和安装 SDK，只负责本地 SDK 发现、索引、激活、取消激活、环境变量、PATH、项目状态和 shell hook。`eget` 只作为可选的 SDK 安装记录来源，不能成为 `xenv` 的硬依赖。

## 目标

1. 去除 `xenv` 配置中下载支持相关字段和示例。
2. 删除或废弃 `tools` 下的下载、安装、更新、卸载类命令。
3. 保留本地 SDK 安装目录规则扫描能力，也就是保留 index 能力。
4. 新增配置项 `eget_enable bool`，仅启用后才优先读取 `eget` 的 SDK 安装记录。
5. 重新整理 CLI 命令结构，让 `xenv` 的定位从“安装工具”变为“激活本地 SDK 和环境”。

## 非目标

1. `xenv` 不实现 SDK 下载。
2. `xenv` 不实现 SDK 更新。
3. `xenv` 不删除 `eget` 安装的 SDK。
4. `xenv` 不修改 `eget` 的配置文件或 installed store。
5. `xenv` 不要求用户必须安装 `eget`。

## 职责边界

### xenv 负责

- 定义本地 SDK 的激活规则。
- 根据配置中的安装目录规则扫描本地 SDK。
- 维护 `~/.xenv/tools.local.json` 本地索引。
- 读取 `.xenv.toml`、global/session state。
- 激活 SDK，向当前 shell 注入 PATH 和环境变量。
- 取消激活 SDK，从当前 shell 移除 PATH 和环境变量。
- 管理 `env`、`path`、`shell`、`config` 相关能力。

### eget 负责

- 下载 SDK。
- 解压 SDK。
- 更新 SDK。
- 删除 SDK。
- 维护自己的 SDK installed store，例如 `~/.config/eget/sdk.installed.json`。
- 管理下载缓存、断点续传、镜像、SDK index cache 等下载相关能力。

## 配置设计

### 新增配置项

```yaml
eget_enable: false
eget_store_file: ""
```

字段含义：

- `eget_enable`: 是否启用 `eget` SDK installed store 作为优先查找来源。默认 `false`。
- `eget_store_file`: 可选，自定义 `eget` SDK installed store 路径。为空时使用默认路径 `~/.config/eget/sdk.installed.json`。

`eget_store_file` 不是必须项，但建议保留。原因是 `eget` 支持通过环境变量改变配置目录；如果用户有自定义配置目录，`xenv` 需要一个稳定、显式的读取路径。

### 保留配置项

```yaml
allow_up_match: 1

sdks:
  - name: go
    alias: golang
    install_dir: "D:/work/env/devsdk/gosdk/go{version}"
    bin_dir: "bin"
    active_env:
      GOROOT: "{install_dir}"

  - name: node
    install_dir: "D:/work/env/devsdk/nodejs/node-v{version}-win-x64"
    bin_dir: ""
```

保留字段：

- `sdks[].name`: SDK 名称。
- `sdks[].alias`: SDK 别名。
- `sdks[].install_dir`: 本地安装目录规则，支持 `{version}`。
- `sdks[].bin_dir`: 激活时追加到 PATH 的相对 bin 目录。
- `sdks[].active_env`: 激活时设置的环境变量，支持 `{name}`、`{version}`、`{install_dir}`。
- `sdks[].other_versions`: 不符合统一目录规则的手动版本目录。
- `allow_up_match`: 版本向上匹配策略。
- `global_env`、`global_paths`、`shell_aliases`、`shell_hooks_dir` 等环境相关配置。

### 删除或废弃配置项

以下字段与下载/安装职责绑定，应从默认配置、文档示例和业务逻辑中移除：

```yaml
install_dir: "~/.xenv/tools"
download_ext:
download_dir:
tools_index_source:
tools_backends:

sdks:
  - install_url: ""
    download_ext: {}
    post_install: []

tools: []
```

说明：

- 顶层 `install_dir` 容易被理解为 `xenv` 的 SDK 下载目标目录，应删除。
- `sdks[].install_dir` 保留，因为它描述的是“本地已安装 SDK 的目录规则”，不是下载目标。
- `tools` 简单工具下载管理应交给 `eget install/add/list/update/uninstall`。
- `post_install` 属于安装生命周期，`xenv` 不再负责。

## SDK 查找模型

`xenv use go:1.22` 的查找顺序：

1. 读取 `xenv` 配置，确认 `go` 是已定义 SDK。
2. 如果 `eget_enable: true`：
   - 读取 `eget_store_file` 指定的 SDK installed store。
   - 如果 `eget_store_file` 为空，读取默认 `~/.config/eget/sdk.installed.json`。
   - 在 store 中查找 SDK 名称和版本。
   - 使用 `eget` 记录中的 `path` 作为 `install_dir`。
   - 仍然使用 `xenv` 配置中的 `bin_dir` 和 `active_env` 渲染 PATH/ENV。
3. 如果未启用 `eget`，或 `eget` 记录中未找到匹配版本：
   - 读取 `~/.xenv/tools.local.json`。
   - 使用当前版本匹配规则匹配 `latest`、`1`、`1.22`、`1.22.0`。
4. 如果仍然找不到：
   - 未启用 `eget` 时提示：`run "xenv sdk index" after installing SDK locally`。
   - 已启用 `eget` 时提示：`install with "eget sdk install go@1.22" or run "xenv sdk index"`。

重要约束：

- `eget_enable: true` 只影响查找优先级。
- `xenv sdk index` 永远只扫描 `xenv` 配置中的本地安装目录规则，不写入 `eget` store。
- `xenv` 不调用 `eget` CLI，不 shell out，不要求 `eget` 在 PATH 中。
- `xenv` 只读取 `eget` 的 JSON store。读取失败时应 fallback 到 xenv 本地索引，并给出 warning。

## 命令结构设计

推荐引入 `sdk` 命名空间，并将 `tools` 作为兼容 alias。

```text
xenv
  use <sdk:version>...
  unuse <sdk>...
  sdk
    index
    list [name]
    show <name>
    where <sdk:version>
    refresh
  env
    set
    get
    unset
    list
  path
    add
    remove
    list
    search
  config
    get
    set
    import
    export
  list
    sdk
    env
    path
    activity
    all
  shell
  init
```

### 高频顶层命令

保留：

```bash
xenv use go:1.22
xenv use -g go:1.22
xenv use -s go:1.22
xenv unuse go
```

原因：

- 激活和取消激活是 `xenv` 的核心高频操作。
- 放在顶层可以减少 shell hook 场景下的输入成本。

### sdk 命令

```bash
xenv sdk index
```

按 `sdks[].install_dir` 和 `sdks[].other_versions` 扫描本地 SDK，生成 `~/.xenv/tools.local.json`。

```bash
xenv sdk list
xenv sdk list go
```

列出本地可激活 SDK。启用 `eget_enable` 时，建议默认显示合并后的视图，并标注来源：

```text
go
  1.22.0  eget   D:/work/env/devsdk/go/go1.22.0
  1.21.5  xenv   D:/work/env/devsdk/go/go1.21.5
```

```bash
xenv sdk show go
```

显示某个 SDK 的配置、本地索引、eget 记录、可激活版本、bin 路径、active env。

```bash
xenv sdk where go:1.22
```

输出匹配到的 SDK 安装目录。可选增加 `--bin` 输出 bin 目录：

```bash
xenv sdk where --bin go:1.22
```

```bash
xenv sdk refresh
```

作为 `xenv sdk index` 的 alias，语义上更适合用户理解“刷新本地可用 SDK 列表”。

### tools 兼容层

短期保留：

```text
xenv tools      alias to xenv sdk
xenv tool       alias to xenv sdk
xenv sdk        primary command
xenv sdks       alias to xenv sdk
```

兼容以下命令：

```bash
xenv tools index
xenv tools list
xenv tools show go
```

不再提供：

```bash
xenv tools install
xenv tools uninstall
xenv tools update
xenv tools register
xenv tools add
```

这些命令的帮助信息不应出现在 `xenv tools --help` 中。

## 与 eget 的用户工作流

默认不启用 `eget`：

```bash
# 用户自己安装 SDK 到配置规则匹配的目录
xenv sdk index
xenv sdk list
xenv use go:1.22
```

启用 `eget`：

```yaml
eget_enable: true
```

```bash
eget sdk install go@1.22.0
eget sdk install node@20.11.1

xenv sdk list
xenv use go:1.22
xenv use -s node:20
```

如果用户不想让 `xenv` 读取 `eget`：

```yaml
eget_enable: false
```

此时即使 `eget` 已安装 SDK，`xenv` 也只使用自己的本地索引。

## 实现建议

### 配置模型

修改 `internal/xenv/models/config.go`：

- 新增 `EgetEnable bool json:"eget_enable"`。
- 新增 `EgetStoreFile string json:"eget_store_file"`。
- 删除或废弃 `DownloadExt`、`DownloadDir`、顶层 `InstallDir`、`Tools`。

删除字段可以分阶段执行：

1. 第一阶段：代码不再读取下载字段，默认配置和文档删除它们。
2. 第二阶段：确认无兼容压力后，从 struct 中删除字段。

### SDK 查找源

建议引入内部接口，避免把 `eget` JSON 读取散落在业务逻辑里：

```go
type SDKSource interface {
	ListSDKVersions(name string) ([]models.InstalledTool, error)
}
```

实现：

- `XenvIndexSource`: 读取 `~/.xenv/tools.local.json`。
- `EgetStoreSource`: 读取 `eget` SDK installed store，并映射为 `models.InstalledTool`。
- `CompositeSDKSource`: 当 `eget_enable` 为 true 时，先查 `EgetStoreSource`，再查 `XenvIndexSource`。

`ToolManager` 或后续改名后的 `SDKManager` 负责版本匹配，不应让 `ToolService` 直接知道 JSON 文件格式。

### CLI 文件

建议把 `internal/cli/tools_cmd.go` 重构为：

```text
internal/cli/sdk_cmd.go
```

命令定义：

- `SDKCmd`
- `SDKIndexCmd`
- `SDKListCmd`
- `SDKShowCmd`
- `SDKWhereCmd`

`SDKCmd` 使用：

```go
Name: "sdk"
Aliases: []string{"sdks", "tool", "tools"}
```

### 服务层

`ToolService` 当前同时包含安装、索引、激活、shell hook 逻辑。建议分阶段收缩：

第一阶段只做删除下载能力：

- 删除 `InstallTool`
- 删除 `UpdateTool`
- 删除 `Uninstall`
- 删除 `Register`
- 保留 `IndexLocalTools`
- 保留 `ListAll`
- 保留 `ActivateSDKs`
- 保留 `DeactivateSDKs`

后续可以再把命名从 `ToolService` 调整为 `SDKService`，但不是第一阶段必须项。

### 删除代码

可以删除或停止使用：

```text
internal/xenv/tools/installer.go
internal/xenv/tools/uninstaller.go
```

保留：

```text
internal/xenv/tools/version.go
```

因为 `ParseVersionSpec` 和版本规格仍然服务于 `use/unuse`。

如果后续想进一步消除命名混淆，可把 `internal/xenv/tools/version.go` 移到 `internal/xenv/sdk/version.go`，但这属于第二阶段重构。

## 测试策略

### 配置测试

- 默认 `eget_enable` 为 false。
- 配置文件可以读取 `eget_enable: true`。
- 配置文件可以读取自定义 `eget_store_file`。
- 删除下载字段后，默认配置示例不再包含下载配置。

### 索引测试

- `xenv sdk index` 仍按 `sdks[].install_dir` 扫描版本目录。
- `other_versions` 仍能被写入本地索引。
- 本地索引文件结构保持兼容。

### eget store 测试

- `eget_enable: true` 时，优先返回 eget store 中的 SDK。
- eget store 不存在时 fallback 到 xenv 本地索引。
- eget store JSON 损坏时 fallback 到 xenv 本地索引，并返回 warning 或可诊断错误。
- eget store 的 `path` 正确映射为 `models.InstalledTool.InstallDir`。

### 命令测试

- `xenv sdk --help` 不显示 install/update/uninstall/register。
- `xenv sdk index` 可运行。
- `xenv tools index` 作为兼容 alias 可运行。
- `xenv use go:1.22` 在只存在 xenv 本地索引时可激活。
- `xenv use go:1.22` 在 `eget_enable: true` 且 eget 有记录时优先使用 eget 路径。

### 回归测试

完成涉及主链路的实现后必须运行：

```bash
go test ./...
```

并做 smoke test：

```bash
go run ./cmd/xenv sdk --help
go run ./cmd/xenv tools --help
go run ./cmd/xenv sdk index
go run ./cmd/xenv sdk list
```

## 迁移说明

### 对已有用户

旧命令：

```bash
xenv tools install go:1.22
xenv tools update go:1.22
xenv tools uninstall go:1.22
```

迁移为：

```bash
eget sdk install go@1.22.0
eget sdk remove go@1.22.0
xenv use go:1.22
```

旧命令：

```bash
xenv tools index
xenv tools list
```

仍可使用，但推荐迁移为：

```bash
xenv sdk index
xenv sdk list
```

### 对配置文件

旧配置中的 `install_url`、`download_ext`、`download_dir`、`tools` 不再生效。用户需要保留或补充的是 `sdks[].install_dir`、`sdks[].bin_dir`、`sdks[].active_env`。

## 推荐实施阶段

### 阶段 1：命令和配置收缩

- 新增 `eget_enable` 和 `eget_store_file` 配置项。
- 新增 `sdk` 命令，保留 `tools` alias。
- 删除 `tools install/update/uninstall/register` 注册。
- 更新 README 和 config 示例。
- 保留现有 xenv 本地 index 行为。

### 阶段 2：eget store 可选读取

- 增加 `EgetStoreSource`。
- `eget_enable: true` 时优先读取 eget SDK installed store。
- `sdk list/show/where/use` 使用合并后的 SDK source。

### 阶段 3：删除下载实现代码

- 删除 `installer.go`、`uninstaller.go` 以及服务层安装方法。
- 清理下载字段和文档残留。
- 如有必要，重命名 `ToolService`/`ToolManager` 为 `SDKService`/`SDKManager`。

## 待确认决策

1. 是否接受 `sdk` 作为主命令名，`tools` 仅作为兼容 alias。
2. `xenv sdk list` 在启用 `eget_enable` 时，是否默认显示 xenv index 与 eget store 的合并结果。
3. `eget_store_file` 是否需要首期实现，还是只实现 `eget_enable` 并固定读取默认路径。
4. 第一阶段是否直接从 struct 删除下载字段，还是先保留字段但不再使用。
