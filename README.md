# xenv

English | [简体中文](README.zh-CN.md)

`xenv` is a local development environment manager for SDK versions, environment variables, and `PATH` entries. It is designed for developers who switch between projects that need different toolchains or local runtime settings.

## Features

- Discover and activate locally installed SDKs, such as Go and Node.js.
- Manage global, project-local, and shell-session environment variables.
- Manage global, project-local, and shell-session `PATH` entries.
- Load project state from `.xenv.toml` when you enter a directory through the shell integration.
- Check active SDKs and project tool requirements.
- Generate shell hooks for bash, zsh, PowerShell, and cmd/clink.

## Install

Install by [Eget](https://github.com/inherelab/eget):

```bash
eget install inhere/xenv
```

Install the latest version with Go:

```bash
go install github.com/inhere/xenv/cmd/xenv@latest
```

## Quick Start

Enable shell integration first. This lets `xenv use`, `xenv set`, and `xenv path add` update the current shell instead of only printing shell script output.

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

Initialize the default configuration explicitly if you want to create it before first use:

```bash
xenv config init
```

The default configuration file is created at:

```text
~/.config/xenv/config.yaml
```

Then index locally installed SDKs and activate a version:

```bash
xenv sdk index
xenv sdk list
xenv use go:latest
```

Save project-local settings to `.xenv.toml`:

```bash
xenv use -s go:1.24
xenv set -s APP_ENV local
xenv path add -s ./bin
```

Check the active environment:

```bash
xenv list activity
xenv check
```

## Shell Integration

Generate shell integration scripts:

```bash
xenv shell --type bash
xenv shell --type zsh
xenv shell --type pwsh
xenv shell --type cmd
```

Install the hook into your shell profile:

```bash
# PowerShell
xenv shell --install -t pwsh --profile $PROFILE.CurrentUserAllHosts

# bash or zsh
xenv shell --install -t $SHELL
```

PowerShell can also load the hook directly:

```powershell
Invoke-Expression (&xenv shell --type pwsh)
# or
xenv shell --type pwsh | Out-String | Invoke-Expression
```

When shell integration is active, `xenv` keeps a shell-session state file and can automatically load matching `.xenv.toml` files when you change directories.

## State Scopes

Most environment-changing commands support three scopes:

| Scope | Flag | Storage | Use case |
| --- | --- | --- | --- |
| Session | none | `~/.config/xenv/session/<session_id>.json` | Temporary changes for the current hooked shell |
| Project | `-s`, `--save`, or `-d` | nearest `.xenv.toml` | Project-specific SDKs, env vars, and paths |
| Global | `-g` or `--global` | `~/.config/xenv/global.toml` | Defaults shared by all projects |

Examples:

```bash
# Current shell session
xenv use go:latest
xenv set APP_ENV local
xenv path add ./bin

# Project .xenv.toml
xenv use -s go:1.24
xenv set -s APP_ENV local
xenv path add -s ./bin

# Global state
xenv use -g go:1.24
xenv set -g GOPROXY https://proxy.golang.org,direct
xenv path add -g ~/.local/bin
```

## Project State

Project state is stored in `.xenv.toml`. The shell hook can load it automatically when you enter the project directory.

Example:

```toml
paths = [
  "./bin",
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

Sections:

- `paths`: Project-local entries added to `PATH`.
- `sdks`: SDK versions activated for the project.
- `envs`: Environment variables loaded for the project.
- `tools`: External tool requirements checked by `xenv check tools`.

## SDK Management

SDKs are configured in `config.yaml`, then indexed from local installation directories.

Common commands:

```bash
xenv sdk index
xenv sdk list
xenv sdk list --all
xenv sdk show go
xenv sdk where go:1.24
xenv sdk where --bin go:1.24
xenv sdk which go:1.24
```

Version specs can use either `name:version` or `name@version`:

```bash
xenv use go:1.24
xenv use node@20
```

When the version is omitted, `latest` is used:

```bash
xenv use go
```

Activate or deactivate multiple SDKs at once:

```bash
xenv use go:1.24 node:20
xenv unuse go:1.24 node:20
```

## Environment Variables

List environment variables managed by `xenv`:

```bash
xenv env
xenv env list
xenv list env
```

Set and unset values:

```bash
xenv env set APP_ENV local
xenv env unset APP_ENV
```

Top-level shortcuts are also available:

```bash
xenv set APP_ENV local
xenv unset APP_ENV
```

Use `-s` for project state or `-g` for global state:

```bash
xenv set -s APP_ENV local
xenv unset -g GOPROXY
```

## PATH Management

List managed `PATH` entries:

```bash
xenv path
xenv path list
xenv list path
```

Add, remove, and search entries:

```bash
xenv path add ./bin
xenv path remove ./bin
xenv path search go
```

Use `-s` for project state or `-g` for global state:

```bash
xenv path add -s ./bin
xenv path add -g ~/.local/bin
```

## Tool Checks

Run all checks:

```bash
xenv check
```

Check active SDK availability:

```bash
xenv check sdk
```

Check project tool requirements from `.xenv.toml`:

```bash
xenv check tools
```

Tool requirements live under `[tools]`:

```toml
[tools]
rg = "*"
golangci-lint = ">=1.60,required"
```

## Configuration

Default files:

```text
~/.config/xenv/config.yaml
~/.config/xenv/global.toml
~/.config/xenv/session/<session_id>.json
~/.config/xenv/sdks.local.json
~/.config/xenv/hooks
```

Show the active configuration summary:

```bash
xenv config
```

Read selected configuration values:

```bash
xenv config get bin_dir
xenv config get shell_hooks_dir
```

Export configuration:

```bash
xenv config export zip
xenv config export json
```

Configuration files support environment variable expansion in values, for example `${HOME}` or `${XENV_SDK_ROOT}`.

Example `config.yaml`:

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

SDK fields:

| Field | Description |
| --- | --- |
| `name` | SDK name used in commands, such as `go` |
| `alias` | Optional alias for display or lookup |
| `install_dir` | SDK installation directory template; `{version}` is replaced by the selected version and is used to strictly match indexed directory names |
| `bin_dir` | SDK binary directory relative to `install_dir` |
| `active_env` | Environment variables exported when the SDK is active |
| `other_versions` | Additional versions that should be considered by the SDK index |

## Command Reference

| Command | Description |
| --- | --- |
| `xenv sdk index` | Scan configured SDK directories and update the local SDK index |
| `xenv sdk list` | List installed SDKs |
| `xenv sdk list --all` | List all configured SDKs, including uninstalled ones |
| `xenv sdk show <name>` | Show details for one SDK |
| `xenv sdk where [--bin] <name:version>` | Print an SDK installation or binary path |
| `xenv use [-g] [-s] <name:version>...` | Activate SDK versions |
| `xenv unuse [-g] [-s] <name:version>...` | Deactivate SDK versions |
| `xenv env list` | List managed environment variables |
| `xenv env set [-g] [-s] <name> <value>` | Set an environment variable |
| `xenv env unset [-g] [-s] <name...>` | Remove environment variables |
| `xenv path list` | List managed `PATH` entries |
| `xenv path add [-g] [-s] <path>` | Add a `PATH` entry |
| `xenv path remove [-g] [-s] <path>` | Remove a `PATH` entry |
| `xenv path search <value>` | Search current `PATH` entries |
| `xenv list activity [-t]` | List active SDKs, env vars, paths, and tool requirements |
| `xenv check` | Run SDK and tool checks |
| `xenv check sdk` | Check active SDK availability |
| `xenv check tools` | Check project tool requirements |
| `xenv shell --type <shell>` | Print shell hook script |
| `xenv shell --install -t <shell>` | Install shell hook into a shell profile |
| `xenv config` | Show configuration summary |
| `xenv config get <name>` | Read a supported configuration value |
| `xenv config export <zip\|json>` | Export configuration |

Aliases:

- `xenv sdks` for `xenv sdk`
- `xenv ls` for `xenv list`
- `xenv e` for `xenv env`
- `xenv p` for `xenv path`
- `xenv cfg` for `xenv config`
- `xenv sdk refresh` or `xenv sdk scan` for `xenv sdk index`
- `xenv sdk which` for `xenv sdk where`
- `xenv path rm` or `xenv path delete` for `xenv path remove`

## Development

Run tests:

```bash
go test ./...
```

Build without release compression:

```bash
go build ./cmd/xenv
```

Build with the project Makefile:

```bash
make build
```

The Makefile uses UPX for compression on supported targets, so `upx` must be available for those build targets.
