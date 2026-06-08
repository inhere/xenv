# xenv

[English](README.md) | 简体中文

`xenv` 是一个本地开发环境管理工具，用于管理 SDK 版本、环境变量和 `PATH` 条目。它适合需要在多个项目之间切换不同工具链、运行时路径和本地环境配置的开发场景。

## 功能介绍

- 发现并激活本地已安装的 SDK，例如 Go、Node.js。
- 管理全局、项目级、当前 Shell 会话级的环境变量。
- 管理全局、项目级、当前 Shell 会话级的 `PATH` 条目。
- 通过 Shell 集成，在进入目录时自动加载项目内的 `.xenv.toml`。
- 检查当前激活的 SDK 和项目声明的工具依赖。
- 为 bash、zsh、PowerShell、cmd/clink 生成 Shell Hook。

## 安装

Install by [Eget](https://github.com/inherelab/eget):

```bash
eget install inhere/xenv
```

使用 Go 安装最新版本：

```bash
go install github.com/inhere/xenv/cmd/xenv@latest
```

## 快速开始

先启用 Shell 集成。启用后，`xenv use`、`xenv set`、`xenv path add` 等命令可以直接影响当前 Shell，而不是只输出 Shell 脚本内容。

Bash:

```bash
eval "$(xenv shell --type bash)"
```

Zsh:

```bash
eval "$(xenv shell --type zsh)"
```

PowerShell:

```powershell
Invoke-Expression (&xenv shell --type pwsh)
```

编辑 `~/.config/xenv/config.yaml`，配置SDK的安装目录等信息。

然后索引本地已安装的 SDK，并激活一个版本：

```bash
xenv sdk index
xenv sdk list
xenv use go:latest
```

把项目级配置保存到 `.xenv.toml`：

```bash
xenv use -s go:1.24
xenv set -s APP_ENV local
xenv path add -s ./bin
```

查看当前激活状态：

```bash
xenv list activity
xenv check
```

## Shell 集成

生成 Shell 集成脚本：

```bash
xenv shell --type bash
xenv shell --type zsh
xenv shell --type pwsh
xenv shell --type cmd
```

把 Hook 安装到 Shell 配置文件：

```bash
# PowerShell
xenv shell --install -t pwsh --profile $PROFILE.CurrentUserAllHosts

# bash 或 zsh
xenv shell --install -t $SHELL
```

PowerShell 也可以直接加载 Hook：

```powershell
Invoke-Expression (&xenv shell --type pwsh)
# 或
xenv shell --type pwsh | Out-String | Invoke-Expression
```

启用 Shell 集成后，`xenv` 会维护当前 Shell 会话状态文件，并可以在切换目录时自动加载匹配的 `.xenv.toml`。

## 状态作用域

大多数会改变环境的命令都支持三种作用域：

| 作用域 | 参数 | 存储位置 | 适用场景 |
| --- | --- | --- | --- |
| 会话级 | 无 | `~/.config/xenv/session/<session_id>.json` | 当前已启用 Hook 的 Shell 中临时生效 |
| 项目级 | `-s`、`--save` 或 `-d` | 最近的 `.xenv.toml` | 项目专属 SDK、环境变量和路径 |
| 全局级 | `-g` 或 `--global` | `~/.config/xenv/global.toml` | 所有项目共享的默认状态 |

示例：

```bash
# 当前 Shell 会话
xenv use go:latest
xenv set APP_ENV local
xenv path add ./bin

# 项目 .xenv.toml
xenv use -s go:1.24
xenv set -s APP_ENV local
xenv path add -s ./bin

# 全局状态
xenv use -g go:1.24
xenv set -g GOPROXY https://proxy.golang.org,direct
xenv path add -g ~/.local/bin
```

## 项目状态

项目状态保存在 `.xenv.toml` 中。启用 Shell Hook 后，进入项目目录时可以自动加载。

示例：

```toml
paths = [
  "./bin",
  "windows:C:/Program Files (x86)/NSIS",
  "linux:/opt/nsis/bin",
  "darwin:/opt/homebrew/bin",
]

[sdks]
go = "1.24"
node = "20"

[envs]
APP_ENV = "local"

[tools]
rg = "*"
golangci-lint = ">=1.60,required"
```

字段说明：

- `paths`: 项目级 `PATH` 条目。可以使用 `windows:`、`linux:`、`darwin:` 前缀限制只在对应系统生效；加入 `PATH` 前会自动移除前缀。
- `sdks`: 项目需要激活的 SDK 版本。
- `envs`: 项目需要加载的环境变量。
- `tools`: `xenv check tools` 会检查的外部工具要求。

## SDK 管理

SDK 先在 `config.yaml` 中声明，再从本地安装目录建立索引。

常用命令：

```bash
xenv sdk index
xenv sdk list
xenv sdk list --all
xenv sdk show go
xenv sdk where go:1.24
xenv sdk where --bin go:1.24
xenv sdk which go:1.24
```

版本规格支持 `name:version` 或 `name@version`：

```bash
xenv use go:1.24
xenv use node@20
```

省略版本时默认使用 `latest`：

```bash
xenv use go
```

也可以一次激活或取消多个 SDK：

```bash
xenv use go:1.24 node:20
xenv unuse go:1.24 node:20
```

## 环境变量管理

列出 `xenv` 管理的环境变量：

```bash
xenv env
xenv env list
xenv list env
```

设置和删除环境变量：

```bash
xenv env set APP_ENV local
xenv env unset APP_ENV
```

也可以使用顶层快捷命令：

```bash
xenv set APP_ENV local
xenv unset APP_ENV
```

使用 `-s` 保存到项目状态，使用 `-g` 保存到全局状态：

```bash
xenv set -s APP_ENV local
xenv unset -g GOPROXY
```

## PATH 管理

列出已管理的 `PATH` 条目：

```bash
xenv path
xenv path list
xenv list path
```

添加、移除和搜索条目：

```bash
xenv path add ./bin
xenv path remove ./bin
xenv path search go
```

使用 `-s` 保存到项目状态，使用 `-g` 保存到全局状态：

```bash
xenv path add -s ./bin
xenv path add -g ~/.local/bin
```

## 工具检查

运行全部检查：

```bash
xenv check
```

检查当前激活的 SDK 是否可用：

```bash
xenv check sdk
```

检查 `.xenv.toml` 中声明的项目工具依赖：

```bash
xenv check tools
```

工具依赖写在 `[tools]` 下：

```toml
[tools]
rg = "*"
golangci-lint = ">=1.60,required"
```

## 配置

默认文件：

```text
~/.config/xenv/config.yaml
~/.config/xenv/global.toml
~/.config/xenv/session/<session_id>.json
~/.config/xenv/sdks.local.json
~/.config/xenv/hooks
```

查看当前配置摘要：

```bash
xenv config
```

读取部分配置项：

```bash
xenv config get bin_dir
xenv config get shell_hooks_dir
```

导出配置：

```bash
xenv config export zip
xenv config export json
```

配置文件中的值支持环境变量展开，例如 `${HOME}` 或 `${XENV_SDK_ROOT}`。

示例 `config.yaml`：

```yaml
bin_dir: "~/.local/bin"
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
    install_dir: "${XENV_SDK_ROOT}/go{version}"
    bin_dir: "bin"
    active_env:
      GOROOT: "{install_dir}"
  - name: node
    install_dir: "${XENV_SDK_ROOT}/node-v{version}"
    bin_dir: "bin"
```

SDK 字段说明：

| 字段 | 说明 |
| --- | --- |
| `name` | 命令中使用的 SDK 名称，例如 `go` |
| `alias` | 可选别名，用于展示或查找 |
| `install_dir` | SDK 安装目录模板；`{version}` 会替换为选中的版本，并用于索引时严格匹配目录名；`{anyword}` 匹配一个非空路径名片段 |
| `bin_dir` | 相对于 `install_dir` 的 SDK 可执行文件目录 |
| `active_env` | 激活 SDK 时导出的环境变量 |
| `other_versions` | 需要加入 SDK 索引的其他版本 |

## 命令索引

| 命令 | 说明 |
| --- | --- |
| `xenv sdk index` | 扫描已配置的 SDK 目录并更新本地 SDK 索引 |
| `xenv sdk list` | 列出已安装 SDK |
| `xenv sdk list --all` | 列出全部已配置 SDK，包括未安装版本 |
| `xenv sdk show <name>` | 查看某个 SDK 的详细信息 |
| `xenv sdk where [--bin] <name:version>` | 输出 SDK 安装目录或二进制目录 |
| `xenv use [-g] [-s] <name:version>...` | 激活 SDK 版本 |
| `xenv unuse [-g] [-s] <name:version>...` | 取消激活 SDK 版本 |
| `xenv env list` | 列出已管理的环境变量 |
| `xenv env set [-g] [-s] <name> <value>` | 设置环境变量 |
| `xenv env unset [-g] [-s] <name...>` | 删除环境变量 |
| `xenv path list` | 列出已管理的 `PATH` 条目 |
| `xenv path add [-g] [-s] <path>` | 添加 `PATH` 条目 |
| `xenv path remove [-g] [-s] <path>` | 删除 `PATH` 条目 |
| `xenv path search <value>` | 搜索当前 `PATH` 条目 |
| `xenv list activity [-t]` | 查看当前激活的 SDK、环境变量、路径和工具要求 |
| `xenv check` | 运行 SDK 和工具检查 |
| `xenv check sdk` | 检查当前激活 SDK 是否可用 |
| `xenv check tools` | 检查项目工具依赖 |
| `xenv shell --type <shell>` | 输出 Shell Hook 脚本 |
| `xenv shell --install -t <shell>` | 安装 Shell Hook 到配置文件 |
| `xenv config` | 查看配置摘要 |
| `xenv config get <name>` | 读取支持的配置项 |
| `xenv config export <zip\|json>` | 导出配置 |

别名：

- `xenv sdks` 等同于 `xenv sdk`
- `xenv ls` 等同于 `xenv list`
- `xenv e` 等同于 `xenv env`
- `xenv p` 等同于 `xenv path`
- `xenv cfg` 等同于 `xenv config`
- `xenv sdk refresh` 或 `xenv sdk scan` 等同于 `xenv sdk index`
- `xenv sdk which` 等同于 `xenv sdk where`
- `xenv path rm` 或 `xenv path delete` 等同于 `xenv path remove`

## 开发

运行测试：

```bash
go test ./...
```

不带发布压缩的构建：

```bash
go build ./cmd/xenv
```

使用项目 Makefile 构建：

```bash
make build
```

> Makefile 会在支持的目标上使用 UPX 压缩，因此运行这些构建目标前需要确保 `upx` 可用。
