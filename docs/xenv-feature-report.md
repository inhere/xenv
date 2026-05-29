# xenv 功能报告

## 1. 模块定位

`xenv` 是一个本地开发环境管理和 SDK 激活工具，当前职责聚焦在：

- 管理本地已安装 SDK 的索引、查询和激活
- 管理环境变量和 `PATH`
- 支持全局、当前 shell 会话、项目目录三种作用域
- 通过 shell hook 让环境变更立即作用于当前终端
- 通过 `.xenv.toml` 管理项目级开发环境声明

当前模块入口位于：

- CLI 命令：`internal/cli`
- 默认配置样例：`config/config.yaml`
- 功能设计：`docs/design/2026-05-29-xenv-sdk-eget-command-design.md`

## 2. 功能总览

### 2.1 SDK 管理

`xenv` 使用 `sdk` 命名空间管理本地 SDK 发现和查询，不再提供 `tools` 命令。

常用命令：

```bash
xenv sdk index
xenv sdk refresh
xenv sdk scan
xenv sdk list
xenv sdk show go
xenv sdk where go:1.22
xenv sdk which go:1.22
xenv use go:1.22
xenv unuse go:1.22
```

说明：

- `index`、`refresh`、`scan` 等价，都会按 `sdks[].install_dir` 扫描本地目录。
- `which` 是 `where` 的别名；默认输出 SDK 安装目录，可配合 `--bin` 输出 bin 目录。
- 本地已安装 SDK 元数据保存在 `~/.config/xenv/sdks.local.json`。

版本规格支持：

```bash
xenv use go
xenv use go:1.22
xenv use go@1.22
xenv use go:latest
```

### 2.2 Tool Requirement 检查

项目级 `.xenv.toml` 可以声明 `[tools]`，用于描述项目依赖的外部 CLI 工具：

```toml
[tools]
rg = "*"
golangci-lint = "latest"
```

检查命令：

```bash
xenv check
xenv check sdk
xenv check tools
```

说明：

- `xenv check` 会同时执行 SDK 检查和工具检查。
- `xenv check tools` 会读取合并后的项目状态，检查工具是否存在，并在显式执行时检查版本。
- `[tools]` 只参与检查，不参与 `sdk index`，也不会写入 `sdks.local.json`。

### 2.3 环境变量管理

支持设置、取消和查看环境变量。

```bash
xenv env list
xenv env set FOO bar
xenv env unset FOO
```

快捷写法：

```bash
xenv set FOO bar
xenv unset FOO
```

作用域参数：

```bash
xenv env set FOO bar
xenv env set -g FOO bar
xenv env set -s FOO bar
xenv env set -d FOO bar
```

### 2.4 PATH 管理

支持添加、删除、搜索、查看 `PATH` 条目。

```bash
xenv path list
xenv path add ./bin
xenv path remove ./bin
xenv path search go
```

### 2.5 Shell Hook 集成

生成 hook：

```bash
xenv shell --type bash
xenv shell --type zsh
xenv shell --type pwsh
```

Bash：

```bash
eval "$(xenv shell --type bash)"
```

PowerShell：

```powershell
Invoke-Expression (& xenv shell --type pwsh)
```

hook 会设置：

```text
XENV_HOOK_SHELL
XENV_SESSION_ID
```

### 2.6 目录级环境

`xenv` 支持当前目录或父目录中的 `.xenv.toml`，用于项目级环境配置。

示例：

```toml
paths = [
  "./bin",
]

[sdks]
go = "1.22"
node = "20"

[envs]
APP_ENV = "local"
DEBUG = "true"

[tools]
rg = "*"
```

保存到目录配置：

```bash
xenv use -s go:1.22
xenv set -s APP_ENV local
xenv path add -s ./bin
```

## 3. 配置文件

默认配置文件：

```text
~/.config/xenv/config.yaml
```

默认配置目录：

```text
~/.config/xenv/
```

默认状态和索引文件：

```text
~/.config/xenv/global.toml
~/.config/xenv/session/<session_id>.json
~/.config/xenv/sdks.local.json
```

默认配置项示例：

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

说明：

- `sdks[].install_dir` 描述本地已安装 SDK 的目录规则。
- `eget_enable` 和 `eget_store_file` 用于是否合并 `eget` 的 installed store 信息。
- `source_project_scripts` 控制是否允许加载项目级 shell 脚本。
- `allow_up_match` 控制版本向上匹配策略。

## 4. 状态模型

`xenv` 的激活状态分三层：

```text
全局状态: ~/.config/xenv/global.toml
目录状态: 当前目录或父目录的 .xenv.toml
会话状态: ~/.config/xenv/session/<session_id>.json
```

加载顺序：

```text
global -> direnv -> session
```

状态内容主要包含：

```toml
paths = []

[sdks]
go = "1.22"

[envs]
APP_ENV = "local"

[tools]
rg = "*"
```

## 5. 推荐使用流程

### 5.1 初始化

```bash
xenv init
```

该命令会初始化配置文件和运行目录。

### 5.2 手动安装 SDK 后纳入管理

配置 `~/.config/xenv/config.yaml`：

```yaml
sdks:
  - name: go
    install_dir: "D:/work/env/devsdk/gosdk/go{version}"
    bin_dir: "bin"
    active_env:
      GOROOT: "{install_dir}"
```

索引本地 SDK：

```bash
xenv sdk index
```

查看 SDK：

```bash
xenv sdk list
```

定位 SDK：

```bash
xenv sdk where go:1.22
xenv sdk which go:1.22
```

激活版本：

```bash
xenv use go:1.22
```

设置全局默认：

```bash
xenv use -g go:1.22
```

设置项目目录专用：

```bash
xenv use -s go:1.22
```

### 5.3 检查项目工具需求

```bash
xenv check tools
```

如果项目定义了 `[tools]`，该命令会检查工具是否存在，以及在显式执行时校验版本信息。

## 6. 命令清单

### 6.1 主命令

```text
sdk
check
use
unuse
env
path
config
list
init
shell
shell-init-hook
shell-direnv
```

### 6.2 sdk

```bash
xenv sdk index
xenv sdk refresh
xenv sdk scan
xenv sdk list
xenv sdk show <name>
xenv sdk where [--bin] <name:version>
xenv sdk which [--bin] <name:version>
```

### 6.3 check

```bash
xenv check
xenv check sdk
xenv check tools
```

### 6.4 use / unuse

```bash
xenv use [-g] [-s|-d] <name:version>...
xenv unuse [-g] [-s|-d] <name:version>...
```

### 6.5 env

```bash
xenv env
xenv env list
xenv env set [-g] [-s|-d] <name> <value>
xenv env unset [-g] [-s|-d] <name...>
```

### 6.6 path

```bash
xenv path
xenv path list
xenv path add [-g] [-s|-d] <path>
xenv path remove [-g] [-s|-d] <path>
xenv path search <value>
```

## 7. 当前实现完成度

可用度较高：

- `xenv init`
- shell hook 脚本生成
- 环境变量设置、删除、查看
- `PATH` 添加、删除、搜索、查看
- `.xenv.toml` 目录状态加载和保存
- 本地 SDK 索引 `sdk index/refresh/scan`
- SDK 查询 `sdk list/show/where/which`
- `check tools`
- 激活本地已索引 SDK `use`

仍需关注：

- `config set`、`config import` 的保存写回逻辑仍需单独验证
- shell profile 自动写入能力仍不应作为主流程依赖
- 项目脚本加载和 direnv 相关行为需要按实际 shell 场景继续验证

## 8. 最小可用示例

独立 CLI：

```bash
go run ./cmd/xenv init
go run ./cmd/xenv sdk list
go run ./cmd/xenv shell --type bash
```

PowerShell：

```powershell
xenv init
Invoke-Expression (& xenv shell --type pwsh)

xenv sdk index
xenv sdk list
xenv use go:latest
xenv check tools
```

Bash：

```bash
xenv init
eval "$(xenv shell --type bash)"

xenv sdk index
xenv sdk list
xenv use go:latest
xenv check tools
```

项目目录专用环境：

```bash
xenv use -s go:1.22
xenv set -s APP_ENV local
xenv path add -s ./bin
```

## 9. 使用建议

现阶段推荐将 `xenv` 用作“本地已安装 SDK 的激活器”和“项目环境变量/PATH 管理器”：

- SDK 先通过系统包管理器、手动安装或 `eget` 管理。
- 在 `config.yaml` 中声明 SDK 安装目录规则。
- 使用 `xenv sdk index` 建立本地索引。
- 使用 `xenv sdk where` / `xenv sdk which` 查询路径。
- 使用 shell hook 中的 `xenv use` 激活版本。
- 使用 `xenv check tools` 检查项目工具依赖。
- 使用 `.xenv.toml` 固化项目级环境。
