# xenv

`xenv` is a local development environment and SDK activator.

It manages:

- SDK/toolchain discovery and activation, such as Go and Node.js
- Environment variables
- `PATH` entries
- Project-local environment state through `.xenv.toml`
- Shell integration for bash, zsh, PowerShell, and cmd/clink

## Install

```bash
go install github.com/inhere/xenv/cmd/xenv@latest
```

## Quick Start

Initialize configuration:

```bash
xenv config init
```

Enable shell integration.

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
Invoke-Expression (& xenv shell --type pwsh)
```

Index locally installed SDKs:

```bash
xenv sdk index
```

Other equivalent commands:

```bash
xenv sdk refresh
xenv sdk scan
```

List configured SDKs:

```bash
xenv sdk list
```

Show or locate a specific SDK:

```bash
xenv sdk where go:1.22
xenv sdk which go:1.22
```

Check project tool requirements from `.xenv.toml`:

```bash
xenv check tools
```

Activate a version:

```bash
xenv use go:latest
xenv use node:20
```

Set environment variables:

```bash
xenv set APP_ENV local
xenv unset APP_ENV
```

Manage `PATH`:

```bash
xenv path add ./bin
xenv path remove ./bin
```

## Project State

Use `-s` or `--save` to save changes to the nearest `.xenv.toml` file:

```bash
xenv use -s go:1.24
xenv set -s APP_ENV local
xenv path add -s ./bin
```

Example `.xenv.toml`:

```toml
paths = [
  "./bin",
]

[sdks]
go = "1.24"
node = "20"

[envs]
APP_ENV = "local"
```

Optional project tool requirements:

```toml
[tools]
rg = "*"
golangci-lint = "latest"
```

## Configuration

Default configuration path:

```text
~/.config/xenv/config.yaml
```

Default state files:

```text
~/.config/xenv/global.toml
~/.config/xenv/session/<session_id>.json
```

Local SDK index:

```text
~/.config/xenv/sdks.local.json
```

## Development

Run tests:

```bash
go test ./...
```

Build:

```bash
go build ./cmd/xenv
```
