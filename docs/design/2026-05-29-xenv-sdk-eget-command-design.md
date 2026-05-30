# xenv SDK 命令与 eget 可选集成设计

## 背景

`xenv` 当前从 `kite-go` 迁移出来后，仍保留了一部分下载、安装、卸载 SDK/工具的设计和代码，例如 `tools install`、`tools update`、`tools uninstall`、`install_url`、`download_ext`、`download_dir` 等。与此同时，工作区内的 `eget` 已经承担了更完整的下载、解压、缓存、SDK 安装记录能力。

新的方向是收缩 `xenv` 的职责：`xenv` 不再负责下载和安装 SDK，只负责本地 SDK 发现、索引、激活、取消激活、环境变量、PATH、项目状态和 shell hook。`eget` 只作为可选的 SDK 安装记录来源，不能成为 `xenv` 的硬依赖。

## 目标

1. 去除 `xenv` 配置中下载支持相关字段和示例。
2. 删除 `tools` 命名空间以及下载、安装、更新、卸载类命令。
3. 保留本地 SDK 安装目录规则扫描能力，也就是保留 index 能力。
4. 新增配置项 `eget_enable bool`，仅启用后才优先读取 `eget` 的 SDK 安装记录。
5. 用户级配置和状态目录统一收敛到 `~/.config/xenv`。
6. 支持 `XENV_CONFIG_DIR` 自定义用户级配置和状态目录。
7. 重新整理 CLI 命令结构，让 `xenv` 的定位从“安装工具”变为“激活本地 SDK 和环境”。

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
- 维护 `~/.config/xenv/sdks.local.json` 本地 SDK 索引。
- 读取 `.xenv.toml`、global/session state。
- 激活 SDK，向当前 shell 注入 PATH 和环境变量。
- 取消激活 SDK，从当前 shell 移除 PATH 和环境变量。
- 管理 `env`、`path`、`shell`、`config` 相关能力。
- 检查项目声明的外部 CLI 工具是否存在或满足版本要求。

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

已确认决策：

- `eget_store_file` 跟随 `eget_enable` 一起实现。
- 当 `eget_enable: true` 且 `eget_store_file` 为空时，读取默认 `~/.config/eget/sdk.installed.json`。
- 当 `eget_store_file` 非空时，优先读取该路径。
- 不做“只实现 eget_enable、固定默认路径”的半成品，因为这会让自定义 eget 配置目录的用户无法完整使用该能力。

### 用户级目录

`xenv` 的用户级配置和状态目录统一使用：

```text
~/.config/xenv/
```

可通过环境变量覆盖：

```bash
export XENV_CONFIG_DIR="$HOME/.config/xenv-dev"
```

PowerShell:

```powershell
$env:XENV_CONFIG_DIR = "$HOME\.config\xenv-dev"
```

最终路径：

```text
~/.config/xenv/config.yaml
~/.config/xenv/global.toml
~/.config/xenv/session/<session_id>.json
~/.config/xenv/sdks.local.json
~/.config/xenv/hooks/
```

当 `XENV_CONFIG_DIR` 存在时，上述路径全部相对该目录计算：

```text
$XENV_CONFIG_DIR/config.yaml
$XENV_CONFIG_DIR/global.toml
$XENV_CONFIG_DIR/session/<session_id>.json
$XENV_CONFIG_DIR/sdks.local.json
$XENV_CONFIG_DIR/hooks/
```

不再使用：

```text
~/.xenv/session/<session_id>.json
~/.xenv/tools.local.json
~/.xenv/sdks.local.json
```

项目级状态文件不迁移，仍然是当前项目目录或父目录中的：

```text
.xenv.toml
```

原因：

- `~/.config/xenv` 是用户级配置和状态目录。
- `XENV_CONFIG_DIR` 便于测试、多配置隔离和临时环境验证。
- `.xenv.toml` 是项目级环境声明文件，应该跟随项目。
- `session`、`global.toml`、`sdks.local.json` 都是用户机器上的运行状态，不应该散落在 `~/.xenv`。

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

### 删除配置项

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
- `tools` 简单工具下载管理应交给 `eget install/add/list/update/uninstall`，`xenv` 中不再保留对应配置。
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
   - 读取 `~/.config/xenv/sdks.local.json`。
   - 使用当前版本匹配规则匹配 `latest`、`1`、`1.22`、`1.22.0`。
4. 如果仍然找不到：
   - 未启用 `eget` 时提示：`run "xenv sdk index" after installing SDK locally`。
   - 已启用 `eget` 时提示：`install with "eget sdk install go@1.22" or run "xenv sdk index"`。

重要约束：

- `eget_enable: true` 只影响查找优先级。
- `xenv sdk index` 永远只扫描 `xenv` 配置中的本地安装目录规则，不写入 `eget` store。
- `xenv` 不调用 `eget` CLI，不 shell out，不要求 `eget` 在 PATH 中。
- `xenv` 只读取 `eget` 的 JSON store。读取失败时应 fallback 到 xenv 本地索引，并给出 warning。

## 项目状态文件 .xenv.toml

`.xenv.toml` 是项目级环境声明文件，不属于用户级配置目录，因此保持在项目根目录或父目录中。它描述当前项目需要的 SDK、环境变量、PATH 和外部 CLI 工具要求。

推荐结构：

```toml
paths = [
  "./bin",
  "./scripts/bin",
]

[sdks]
go = "1.24"
node = "20"

[envs]
APP_ENV = "local"

[tools]
rg = "*"
buf = ">=1.32,required"
golangci-lint = ">=1.60"
protoc = ">=25,required"
```

字段语义：

- `paths`: 项目需要注入 PATH 的目录。
- `[sdks]`: 项目需要激活的 SDK 版本，由 `xenv` 解析并注入 PATH/ENV。
- `[envs]`: 项目需要注入的环境变量。
- `[tools]`: 项目依赖的外部 CLI 工具要求。`xenv` 只检查，不安装、不更新、不激活、不写 PATH。

### tools 字段语义

`.xenv.toml [tools]` 不再表示“由 xenv 管理或安装的工具”，而表示“项目依赖的外部命令检查项”。

规则：

```toml
[tools]
rg = "*"
buf = ">=1.32,required"
golangci-lint = ">=1.60"
protoc = ">=25,optional"
```

含义：

- `"*"`: 只检查命令是否存在。
- `">=1.32"`: 检查命令存在，并检查版本满足 `>=1.32`。默认按 required 处理。
- `">=1.32,required"`: 缺失或版本不满足时返回错误。
- `">=1.32,optional"`: 缺失或版本不满足时返回 warning。

第一版只需要支持：

- `*`
- `>=x`
- `>=x.y`
- `>=x.y.z`
- `required`
- `optional`

不需要第一版支持完整 semver range，例如 `~1.2`、`^1.2`、`>1 <2`。

版本获取默认执行：

```text
<tool> --version
```

版本解析失败时：

- `xenv check tools` 输出 warning。
- 不阻止 SDK/env/path 激活。
- required 工具如果命令存在但版本输出无法解析，第一版按 warning 处理，避免不同工具版本输出格式差异导致项目进入目录失败。

后续如需要更精细控制，可扩展 table 写法：

```toml
[tools.buf]
version = ">=1.32"
required = true
command = "buf"
version_cmd = "buf --version"
install_hint = "eget install --add --name buf bufbuild/buf"
```

但第一版只实现简单 map，避免把 `.xenv.toml` 复杂化。

已确认决策：

- `.xenv.toml [tools]` 第一版只支持简单 map。
- 第一版不支持 `[tools.<name>]` table 写法。
- table 写法作为后续扩展保留，不进入第一版实现范围。

### tools 与 eget 的关系

`.xenv.toml [tools]` 不依赖 `eget_enable`。

原因：

- `eget_enable` 只控制 SDK 查找来源。
- `[tools]` 是项目外部 CLI 工具需求声明。
- 用户可以通过 `eget`、`brew`、`scoop`、`apt`、手动安装等方式满足工具要求。

`xenv` 可以输出安装提示，但不执行安装命令：

```text
missing required tool: buf >=1.32
hint: eget install --add --name buf bufbuild/buf
```

安装提示不是第一阶段必须能力。如果需要，可后续通过全局配置维护：

```yaml
tool_install_hints:
  rg: "eget install --add --name rg BurntSushi/ripgrep"
  buf: "eget install --add --name buf bufbuild/buf"
```

### ActivityState 结构

当前 `ActivityState` 中的 `Tools map[string]string` 应改名，避免继续误解成“已激活工具”或“由 xenv 管理的工具”。

建议改为：

```go
type ActivityState struct {
	Paths []string `json:"paths" toml:"paths"`
	SDKs  map[string]string `json:"sdks" toml:"sdks"`
	Envs  map[string]string `json:"envs" toml:"envs"`

	// ToolRequirements declares external CLI tools required by this state.
	// xenv only checks them; it does not install, update, activate, or remove them.
	ToolRequirements map[string]string `json:"tools" toml:"tools"`
}
```

行为约束：

- 可从 `.xenv.toml [tools]` 读取。
- 可被 global/session/direnv state 合并，供 `xenv check` 使用。
- 不生成 shell script。
- 不写 PATH。
- 不调用 `eget`。
- 不参与 `xenv sdk index`。
- 不写入 `sdks.local.json`。

## 命令结构设计

命令结构只保留 `sdk` 命名空间，不再保留 `tools` 兼容命令。`xenv` 还处于本地开发测试阶段，不需要兼容旧命令。

```text
xenv
  use <sdk:version>...
  unuse <sdk>...
  sdk
    index
    refresh
    scan
    list [name]
    show <name>
    where <sdk:version>
    which <sdk:version>
  check
    sdk
    tools
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
xenv sdk refresh
xenv sdk scan
```

三者等价，按 `sdks[].install_dir` 和 `sdks[].other_versions` 扫描本地 SDK，生成 `~/.config/xenv/sdks.local.json`。

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

已确认决策：

- `eget_enable: false` 时，`xenv sdk list` 只显示 `xenv` 本地索引。
- `eget_enable: true` 时，`xenv sdk list` 默认显示 `xenv` 本地索引与 `eget` SDK installed store 的合并结果。
- 合并结果必须标注来源，例如 `xenv`、`eget`。
- 如果同一 SDK 同一版本同时存在于两个来源，优先显示 `eget` 记录，或合并为一条并标注 `eget,xenv`。第一版建议优先显示 `eget`，避免重复版本干扰用户。

```bash
xenv sdk show go
```

显示某个 SDK 的配置、本地索引、eget 记录、可激活版本、bin 路径、active env。

```bash
xenv sdk where go:1.22
xenv sdk which go:1.22
```

二者等价，输出匹配到的 SDK 安装目录。可选增加 `--bin` 输出 bin 目录：

```bash
xenv sdk where --bin go:1.22
xenv sdk which --bin go:1.22
```

`which` 是 `where` 的 alias，不引入新行为。

`refresh` 强调刷新本地可用 SDK 列表，`scan` 强调重新扫描安装目录。二者都是 `index` 的别名，不引入新行为。

### 删除 tools 命名空间

不再提供以下命令：

```bash
xenv tools
xenv tool
xenv tools index
xenv tools list
xenv tools show
xenv tools install
xenv tools uninstall
xenv tools update
xenv tools register
xenv tools add
```

所有 SDK 发现和索引能力统一放到 `xenv sdk` 下。

### check 命令

新增：

```bash
xenv check
xenv check sdk
xenv check tools
```

`xenv check` 检查当前目录合并后的环境声明：

- `.xenv.toml` 是否存在。
- `[sdks]` 中版本是否可解析、可匹配、可激活。
- `[tools]` 中外部 CLI 工具是否存在，版本是否满足要求。
- `paths` 中的路径是否存在。
- `[envs]` 是否有效。

`xenv check sdk` 只检查 SDK。

`xenv check tools` 只检查工具要求。

输出示例：

```text
SDKs
  OK go 1.24 -> D:/work/env/devsdk/gosdk/go1.24.2
  OK node 20 -> D:/work/env/devsdk/nodejs/node-v20.11.1-win-x64

Tools
  OK rg 14.1.1
  MISSING buf >=1.32 required
    hint: eget install --add --name buf bufbuild/buf
  WARN protoc version output not recognized
```

### shell hook 中的 tools 检查

shell hook 每次进入目录时不应默认执行完整工具版本检查，因为多个 `<tool> --version` 会拖慢 `cd`。

新增配置项：

```yaml
check_tools_on_direnv: false
```

默认行为：

- `xenv init-direnv` 只做 SDK/env/path 激活和可选项目脚本 source 表达式生成。
- 不执行 `[tools]` 版本检查。
- 用户主动执行 `xenv check tools` 时才做完整检查。

如果启用：

```yaml
check_tools_on_direnv: true
```

进入目录时只做轻量检查：

- 用 `exec.LookPath` 检查工具是否存在。
- 不默认执行 `<tool> --version`。
- 缺失 required 工具时输出 warning。
- 不阻止 SDK/env/path 激活。

### 项目脚本自动 source

当前 shell hook 已经设计为加载用户级 hooks 目录：

```text
~/.config/xenv/hooks/*.sh
~/.config/xenv/hooks/*.ps1
```

当前代码中也有 `.envrc` / `.envrc.ps1` 的检测占位，但未看到自动 source 项目目录下 `.xenv.sh` / `.xenv.ps1` 的完整实现。因此设计上将该能力作为显式、可配置的项目脚本加载功能，而不是默认无条件执行。

新增配置项：

```yaml
source_project_scripts: false
```

支持的项目脚本文件：

```text
.xenv.sh   # bash/zsh
.xenv.ps1  # PowerShell
```

行为：

- 默认 `source_project_scripts: false`，不自动 source 项目脚本。
- 启用后，shell hook 在进入目录时调用 `xenv init-direnv`，由 `init-direnv` 根据最近 `.xenv.toml` 所在目录决定是否返回项目脚本 source 表达式。
- shell hook 不直接查找 `.xenv.sh` 或 `.xenv.ps1`，只负责 eval `init-direnv` 返回的表达式。
- bash/zsh 只 source `.xenv.sh`。
- PowerShell 只 dot-source `.xenv.ps1`。
- 不跨 shell 执行脚本，例如 bash 不读取 `.xenv.ps1`。
- 只 source 与当前项目状态文件同目录的脚本，避免向上递归时加载到不相关父目录脚本。
- 如果没有 `.xenv.toml`，不自动 source `.xenv.sh/.xenv.ps1`。这样可以把项目脚本执行和显式 xenv 项目声明绑定起来。

安全约束：

- 第一次发现项目脚本时应提示风险，或至少在 debug/warn 中明确显示正在 source 的文件路径。
- 不建议默认开启，因为 source 项目脚本等价于执行项目代码。
- 项目脚本加载只属于 shell hook 行为，不应由普通 `xenv check`、`xenv sdk list` 等命令触发。

与 `.envrc` 的关系：

- `.envrc` / `.envrc.ps1` 属于 direnv 生态兼容项，可以后续单独设计。
- `.xenv.sh` / `.xenv.ps1` 是 xenv 自己的项目脚本约定，优先级应低于 `xenv` 生成的 SDK/env/path 激活脚本。
- 如果同时存在 `.xenv.toml` 和 `.xenv.sh`，先应用 `.xenv.toml` 中的 SDK/env/path，再 source `.xenv.sh`，让项目脚本可以做最后补充。

命名决策：

- xenv 自有项目脚本使用 `.xenv.sh` / `.xenv.ps1`。
- `.envrc` / `.envrc.ps1` 不作为 xenv 的主命名。
- 原因是 `.envrc` 已经是 direnv 的强约定，用户会预期由 direnv 管理；xenv 如果默认接管 `.envrc`，容易和 direnv 行为、信任模型、allow 机制混淆。
- 后续如确实需要兼容 `.envrc`，应通过单独配置开启，例如 `source_envrc: false`，并明确文档说明它是兼容模式。

已确认决策：

- `source_project_scripts` 默认关闭。
- 只有配置 `source_project_scripts: true` 后才允许 shell hook source `.xenv.sh` / `.xenv.ps1`。
- 即使启用，也要求同目录存在 `.xenv.toml`，否则不 source 项目脚本。

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

### 本地索引结构

本地索引文件改为：

```text
~/.config/xenv/sdks.local.json
```

已确认决策：

- 本地索引文件名确定为 `sdks.local.json`。
- 完整默认路径为 `~/.config/xenv/sdks.local.json`。
- 如果设置 `XENV_CONFIG_DIR`，路径为 `$XENV_CONFIG_DIR/sdks.local.json`。

索引结构只存 SDK，不再预留 `tools` 字段：

```json
{
  "schema": 1,
  "created_at": "2026-05-29T10:00:00+08:00",
  "updated_at": "2026-05-29T10:00:00+08:00",
  "sdks": [
    {
      "id": "go:1.22.0",
      "name": "go",
      "version": "1.22.0",
      "install_dir": "D:/work/env/devsdk/gosdk/go1.22.0",
      "source": "xenv",
      "created_at": "2026-05-29T10:00:00+08:00",
      "updated_at": "2026-05-29T10:00:00+08:00"
    }
  ]
}
```

建议 Go 结构：

```go
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
}
```

说明：

- `source` 对本地扫描结果写为 `xenv`。
- 从 `eget` store 映射出的运行时记录可以使用 `source: "eget"`，但不写入 `sdks.local.json`。
- `bin_dir`、`active_env`、`alias` 不写入本地索引，仍从 `xenv` 配置读取。
- 不保留旧的 `ToolsLocal.Tools` 字段。
- 不保留旧的 `is_sdk` 字段，因为文件已是 SDK 专用索引。

### 配置模型

修改 `internal/xenv/models/config.go`：

- 新增 `EgetEnable bool json:"eget_enable"`。
- 新增 `EgetStoreFile string json:"eget_store_file"`。
- 新增 `CheckToolsOnDirenv bool json:"check_tools_on_direnv"`。
- 新增 `SourceProjectScripts bool json:"source_project_scripts"`。
- 删除 `DownloadExt`、`DownloadDir`、顶层 `InstallDir`、`Tools`。
- 删除 `ToolChain.InstallURL`、`ToolChain.DownloadExt`、`ToolChain.PostInstall`。

配置目录解析新增：

- 优先读取 `XENV_CONFIG_DIR`。
- 如果 `XENV_CONFIG_DIR` 为空，使用 `~/.config/xenv`。
- `config.yaml`、`global.toml`、`session`、`sdks.local.json`、`hooks` 都从同一个配置目录派生。
- `XENV_CONFIG_DIR` 应在测试中可注入，避免测试写入真实用户目录。

### SDK 查找源

建议引入内部接口，避免把 `eget` JSON 读取散落在业务逻辑里：

```go
type SDKSource interface {
	ListSDKVersions(name string) ([]models.InstalledSDK, error)
}
```

实现：

- `XenvIndexSource`: 读取 `~/.config/xenv/sdks.local.json`。
- `EgetStoreSource`: 读取 `eget` SDK installed store，并映射为 `models.InstalledSDK`。
- `CompositeSDKSource`: 当 `eget_enable` 为 true 时，先查 `EgetStoreSource`，再查 `XenvIndexSource`。

`ToolManager` 或后续改名后的 `SDKManager` 负责版本匹配，不应让 `ToolService` 直接知道 JSON 文件格式。

### CLI 文件

建议删除 `internal/cli/tools_cmd.go`，新增：

```text
internal/cli/sdk_cmd.go
```

命令定义：

- `SDKCmd`
- `SDKIndexCmd`
- `SDKListCmd`
- `SDKShowCmd`
- `SDKWhereCmd`

`SDKCmd` 使用 `Name: "sdk"`，不设置 `tools`、`tool`、`sdks` alias。

### 服务层

`ToolService` 当前同时包含安装、索引、激活、shell hook 逻辑。建议分阶段收缩：

- 删除 `InstallTool`
- 删除 `UpdateTool`
- 删除 `Uninstall`
- 删除 `Register`
- 将 `IndexLocalTools` 重命名为 `IndexLocalSDKs`
- 将 `ListAll` 重命名为 `ListSDKs`
- 保留 `ActivateSDKs`
- 保留 `DeactivateSDKs`

同时将命名从 `ToolService`/`ToolManager` 收敛为 `SDKService`/`SDKManager`。由于不考虑兼容性，建议一次性完成命名清理，避免继续传播 `tool` 概念。

已确认决策：

- 接受一次性重命名 `ToolService`/`ToolManager` 为 `SDKService`/`SDKManager`。
- 不保留旧命名兼容层。

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

建议把 `internal/xenv/tools/version.go` 移到 `internal/xenv/sdk/version.go`，因为 `ParseVersionSpec` 和版本规格服务于 SDK 激活，不再属于工具安装模块。

## 测试策略

### 配置测试

- 默认 `eget_enable` 为 false。
- 配置文件可以读取 `eget_enable: true`。
- 配置文件可以读取自定义 `eget_store_file`。
- 默认 `check_tools_on_direnv` 为 false。
- 默认 `source_project_scripts` 为 false。
- `XENV_CONFIG_DIR` 可以覆盖用户级配置目录。
- `XENV_CONFIG_DIR` 为空时默认使用 `~/.config/xenv`。
- 删除下载字段后，默认配置示例不再包含下载配置。
- 用户级状态文件默认写入 `~/.config/xenv`。

### 索引测试

- `xenv sdk index` 仍按 `sdks[].install_dir` 扫描版本目录。
- `other_versions` 仍能被写入本地索引。
- 本地索引文件写入 `~/.config/xenv/sdks.local.json`。
- 本地索引文件只包含 `sdks`，不包含 `tools`。

### eget store 测试

- `eget_enable: true` 时，优先返回 eget store 中的 SDK。
- eget store 不存在时 fallback 到 xenv 本地索引。
- eget store JSON 损坏时 fallback 到 xenv 本地索引，并返回 warning 或可诊断错误。
- eget store 的 `path` 正确映射为 `models.InstalledSDK.InstallDir`。

### 命令测试

- `xenv sdk --help` 不显示 install/update/uninstall/register。
- `xenv sdk index` 可运行。
- `xenv sdk refresh` 行为等价于 `xenv sdk index`。
- `xenv sdk scan` 行为等价于 `xenv sdk index`。
- `xenv sdk which` 行为等价于 `xenv sdk where`。
- `xenv tools --help` 应返回未知命令。
- `xenv check tools` 可检查 `.xenv.toml [tools]`。
- `xenv use go:1.22` 在只存在 xenv 本地索引时可激活。
- `xenv use go:1.22` 在 `eget_enable: true` 且 eget 有记录时优先使用 eget 路径。
- `source_project_scripts: true` 时，`xenv init-direnv` 会在最近 `.xenv.toml` 同目录存在项目脚本时返回 source 表达式，shell hook eval 该表达式。
- `source_project_scripts: false` 时，`xenv init-direnv` 不返回项目脚本 source 表达式。

### 回归测试

完成涉及主链路的实现后必须运行：

```bash
go test ./...
```

并做 smoke test：

```bash
go run ./cmd/xenv sdk --help
go run ./cmd/xenv tools --help # expected: unknown command
go run ./cmd/xenv sdk index
go run ./cmd/xenv sdk refresh
go run ./cmd/xenv sdk scan
go run ./cmd/xenv sdk list
go run ./cmd/xenv sdk which go:1.22
go run ./cmd/xenv check tools
```

## 推荐实施阶段

### 阶段 1：命令和配置收缩

- 新增 `eget_enable` 和 `eget_store_file` 配置项。
- 新增 `check_tools_on_direnv` 配置项。
- 新增 `source_project_scripts` 配置项。
- 新增 `XENV_CONFIG_DIR` 配置目录覆盖。
- 新增 `sdk` 命令。
- 新增 `check` 命令。
- 删除 `tools` 命名空间。
- 更新 README 和 config 示例。
- 将用户级状态目录统一改为 `~/.config/xenv`。
- 将本地索引改为 `~/.config/xenv/sdks.local.json`。
- 将本地索引结构改为 SDK 专用结构。

### 阶段 2：eget store 可选读取

- 增加 `EgetStoreSource`。
- `eget_enable: true` 时优先读取 eget SDK installed store。
- `sdk list/show/where/use` 使用合并后的 SDK source。

### 阶段 3：删除下载实现代码

- 删除 `installer.go`、`uninstaller.go` 以及服务层安装方法。
- 清理下载字段和文档残留。
- 重命名 `ToolService`/`ToolManager` 为 `SDKService`/`SDKManager`。

## 待确认决策

当前设计决策已确认完毕，后续进入实现计划前如发现新的风险，再单独补充。
