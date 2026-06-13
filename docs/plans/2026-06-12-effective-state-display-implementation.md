# xenv Status Command State Semantics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前状态诊断从顶层 `list/ls` 收敛到一等 `xenv status` 命令，清晰展示 Effective State、Session Context，并为 Runtime State 检测保留明确入口。

**Architecture:** v0 阶段不保留顶层 `list/ls` 兼容入口；状态诊断由 `internal/cli/status_cmd.go` 承载，SDK/ENV/PATH 的枚举能力继续保留在 `sdk list`、`env list`、`path list`。第一阶段实现 `xenv status`、`xenv status --layers`、Effective State 计算、`Session Context` 展示、overridden 标注，以及目录内普通 `xenv use` 的临时 runtime override warning；Runtime State 的完整 PATH 检测作为第二阶段，通过 `xenv status --runtime` 承载。

**Tech Stack:** Go, `gookit/gcli/v3`, `github.com/gookit/goutil/x/assert`, TOML/JSON state files, PowerShell/bash/zsh hook script generation.

---

## 修订记录

| 日期 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-12 | v0.1 | Codex | 初版实施计划，聚焦 `ls --group` 的 Effective State 展示和 Runtime 检测评估。 |
| 2026-06-12 | v0.2 | Codex | 补充状态语义设计文档链接，明确 Session State 是 Session Defaults/Context，不是 Runtime State。 |
| 2026-06-13 | v0.3 | Codex | 按 v0 最优方案重写计划：状态诊断收敛到一等 `xenv status`，移除顶层 `list/ls` 入口。 |

## 关联文档

- 设计文档: [../design/2026-06-12-xenv-state-semantics-design.md](../design/2026-06-12-xenv-state-semantics-design.md)

## 背景

原计划以 `xenv ls --group` 承载状态诊断。但在状态语义明确后，`list/ls` 的职责过宽：

```text
list sdk   本机 SDK inventory
list env   xenv 管理的 ENV 状态
list path  xenv 管理的 PATH 状态
list state global/session/direnv 状态诊断
```

状态诊断现在需要表达：

- `Effective State`: 当前目录期望状态。
- `Session Context`: 当前 shell 的 session defaults / recovery baseline。
- `Runtime State`: 当前 shell 实际 PATH/ENV 检测结果。
- `Warnings`: runtime 与 effective 不一致、普通 `use` 在目录状态下形成临时 override 等。

这些语义不再适合放在 `list` 下。`xenv` 仍处于 v0 开发阶段，因此直接移除顶层 `list/ls`，用一等 `xenv status` 做状态诊断主入口。

## 目标命令结构

保留：

```text
xenv use
xenv unuse
xenv env
xenv path
xenv sdk
xenv status
xenv check
xenv shell
xenv config
xenv set
xenv unset
```

移除顶层入口：

```text
xenv list
xenv ls
xenv list state
xenv list all
xenv ls --group
```

保留列表能力，但放在明确的领域命令下：

```text
xenv sdk list
xenv env list
xenv path list
```

## 状态语义

Effective State 优先级：

```text
Directory State > Session State > Global State
```

展示命名：

```text
[Session Context]
Session Defaults:
          go => 1.24.6  (overridden by Directory State)
```

Runtime State 不从 session JSON 直接得出。第一阶段只预留 `xenv status --runtime` 入口和输出骨架，完整 PATH-based 检测放第二阶段。

## 文件结构

计划修改或创建这些文件：

- Create: `internal/cli/status_cmd.go`
  - 承载 `xenv status` 命令、Effective State 纯计算、status 输出格式化。
- Create: `internal/cli/status_cmd_test.go`
  - 覆盖 Effective State 计算、overridden 标注、status 输出格式。
- Modify: `internal/cli/app.go`
  - 注册 `StatusCmd()`，移除 `NewListCmd()`。
- Modify: `internal/cli/list_cmd.go`
  - 删除或收缩顶层 `list` 命令相关代码。若仍有共享 helper，被移动到 `status_cmd.go` 或各领域命令文件。
- Modify: `internal/cli/app_test.go`
  - 更新顶层命令注册测试：不再期望 `list`，新增 `status`。
- Modify: `internal/cli/use_cmd.go`
  - session scope 的 `xenv use` 与 Directory State 冲突时输出临时 runtime override warning。
- Modify: `internal/xenv/shell/gen_hook_bash.go`
  - completion 中移除 `list`，加入 `status` / `st`。
- Modify: `internal/xenv/shell/gen_hook_zsh.go`
  - completion 中移除 `list`，加入 `status` / `st`。
- Modify: `internal/xenv/shell/gen_hook_pwsh.go`
  - completion 中移除 `list`，加入 `status` / `st`。
- Modify: `internal/xenv/shell/gen_hook_test.go`
  - 更新 completion 断言。
- Modify: `README.md`
  - 更新命令组织和状态语义说明。
- Modify: `README.zh-CN.md`
  - 更新中文命令组织和状态语义说明。
- Modify: `docs/design/2026-06-12-xenv-state-semantics-design.md`
  - 若实施中发现细节偏差，同步更新设计决策。
- Modify: `docs/plans/2026-06-12-effective-state-display-implementation.md`
  - 本实施计划。

## Task 1: 注册 `status` 并移除顶层 `list`

**Files:**
- Modify: `internal/cli/app_test.go`
- Modify: `internal/cli/app.go`
- Create: `internal/cli/status_cmd.go`
- Test: `go test ./internal/cli -run TestNewAppRegistersTopLevelCommands -count=1 -v`

- [ ] **Step 1: 更新失败测试**

修改 `internal/cli/app_test.go` 中 `TestNewAppRegistersTopLevelCommands` 的命令列表：

```go
	for _, name := range []string{
		"sdk",
		"check",
		"use",
		"unuse",
		"env",
		"path",
		"config",
		"status",
		"shell",
		"shell-init-hook",
		"shell-direnv",
	} {
		if !app.HasCommand(name) {
			t.Fatalf("expected app to register command %q", name)
		}
	}

	if app.HasCommand("list") {
		t.Fatalf("list command must not be registered at top level; use status or sdk/env/path list")
	}
	if app.ResolveAlias("ls") == "list" {
		t.Fatalf("ls alias must not resolve to removed top-level list command")
	}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/cli -run TestNewAppRegistersTopLevelCommands -count=1 -v
```

Expected:

```text
FAIL
expected app to register command "status"
```

- [ ] **Step 3: 新增最小 `StatusCmd`**

创建 `internal/cli/status_cmd.go`：

```go
package cli

import (
	"fmt"

	"github.com/gookit/gcli/v3"
)

type statusOptions struct {
	Layers  bool `flag:"desc=Show Global, Directory, and Session Context layers"`
	Runtime bool `flag:"desc=Show runtime detection details"`
}

func StatusCmd() *gcli.Command {
	var opts statusOptions
	return &gcli.Command{
		Name:    "status",
		Desc:    "Show current xenv status for this shell and directory",
		Aliases: []string{"st"},
		Config: func(c *gcli.Command) {
			c.MustFromStruct(&opts)
		},
		Func: func(c *gcli.Command, args []string) error {
			fmt.Println("[Effective State]")
			fmt.Println("No effective state found")
			return nil
		},
	}
}
```

- [ ] **Step 4: 更新 app 注册**

修改 `internal/cli/app.go`：

```go
	app.Add(
		SDKCmd,
		CheckCmd,
		NewUseCmd(),
		NewUnuseCmd(),
		EnvCmd,
		PathCmd,
		ConfigCmd,
		StatusCmd(),
		NewShellCmd(),
		ShellHookInitCmd(),
		ShellDirenvCmd(),
	)
```

移除 `NewListCmd()` 注册。

- [ ] **Step 5: 运行测试确认通过**

Run:

```bash
go test ./internal/cli -run TestNewAppRegistersTopLevelCommands -count=1 -v
```

Expected:

```text
PASS
```

- [ ] **Step 6: 提交**

Run:

```bash
git add internal/cli/app.go internal/cli/app_test.go internal/cli/status_cmd.go
git commit -m "feat: add top-level status command"
```

## Task 2: Effective State 纯计算

**Files:**
- Modify: `internal/cli/status_cmd.go`
- Create: `internal/cli/status_cmd_test.go`
- Test: `go test ./internal/cli -run TestBuildEffectiveSDKRows -count=1 -v`

- [ ] **Step 1: 写失败测试**

创建 `internal/cli/status_cmd_test.go`：

```go
package cli

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/xenv/internal/xenv/models"
)

func TestBuildEffectiveSDKRows(t *testing.T) {
	global := models.NewActivityState("global.toml")
	global.SDKs["go"] = "1.21.13"
	global.SDKs["java"] = "17.0.11"

	session := models.NewActivityState("session.json")
	session.SDKs["go"] = "1.24.6"
	session.SDKs["node"] = "22.0.0"

	dir := models.NewActivityState(".xenv.toml")
	dir.SDKs["go"] = "1.25"

	rows := buildEffectiveSDKRows(global, session, []*models.ActivityState{dir})

	assert.Require(t, assert.Eq(t, 3, len(rows)))

	t.Run("directory overrides session and global", func(t *testing.T) {
		assert.Eq(t, "go", rows[0].Name)
		assert.Eq(t, "1.25", rows[0].Version)
		assert.Eq(t, sourceDirectory, rows[0].Source)
		assert.Eq(t, []stateValue{
			{Source: sourceSession, Version: "1.24.6"},
			{Source: sourceGlobal, Version: "1.21.13"},
		}, rows[0].Overrides)
	})

	t.Run("global remains effective when no higher value exists", func(t *testing.T) {
		assert.Eq(t, "java", rows[1].Name)
		assert.Eq(t, "17.0.11", rows[1].Version)
		assert.Eq(t, sourceGlobal, rows[1].Source)
	})

	t.Run("session remains effective when no directory value exists", func(t *testing.T) {
		assert.Eq(t, "node", rows[2].Name)
		assert.Eq(t, "22.0.0", rows[2].Version)
		assert.Eq(t, sourceSession, rows[2].Source)
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/cli -run TestBuildEffectiveSDKRows -count=1 -v
```

Expected:

```text
FAIL
undefined: buildEffectiveSDKRows
```

- [ ] **Step 3: 实现纯计算函数**

在 `internal/cli/status_cmd.go` 中加入：

```go
const (
	sourceGlobal    = "Global State"
	sourceSession   = "Session Context"
	sourceDirectory = "Directory State"
)

type stateValue struct {
	Source  string
	Version string
}

type effectiveSDKRow struct {
	Name      string
	Version   string
	Source    string
	Overrides []stateValue
}
```

添加 import：

```go
import (
	"fmt"
	"sort"

	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/xenv/models"
)
```

添加函数：

```go
func buildEffectiveSDKRows(global, session *models.ActivityState, dirStates []*models.ActivityState) []effectiveSDKRow {
	values := make(map[string][]stateValue)

	collectSDKValues(values, sourceGlobal, global)
	collectSDKValues(values, sourceSession, session)
	for _, dirState := range dirStates {
		collectSDKValues(values, sourceDirectory, dirState)
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([]effectiveSDKRow, 0, len(names))
	for _, name := range names {
		stack := values[name]
		effective := stack[len(stack)-1]
		overrides := make([]stateValue, 0, len(stack)-1)
		for i := len(stack) - 2; i >= 0; i-- {
			if stack[i].Version != effective.Version {
				overrides = append(overrides, stack[i])
			}
		}
		rows = append(rows, effectiveSDKRow{
			Name:      name,
			Version:   effective.Version,
			Source:    effective.Source,
			Overrides: overrides,
		})
	}
	return rows
}

func collectSDKValues(values map[string][]stateValue, source string, state *models.ActivityState) {
	if state == nil {
		return
	}
	for name, version := range state.SDKs {
		values[name] = append(values[name], stateValue{Source: source, Version: version})
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./internal/cli -run TestBuildEffectiveSDKRows -count=1 -v
```

Expected:

```text
PASS
```

- [ ] **Step 5: 提交**

Run:

```bash
git add internal/cli/status_cmd.go internal/cli/status_cmd_test.go
git commit -m "test: cover effective status rows"
```

## Task 3: `status` 输出 Effective State 和 Session Context

**Files:**
- Modify: `internal/cli/status_cmd.go`
- Modify: `internal/cli/status_cmd_test.go`
- Test: `go test ./internal/cli -run "TestFormatEffectiveSDKRows|TestFormatSessionContextLines" -count=1 -v`

- [ ] **Step 1: 写格式化测试**

在 `internal/cli/status_cmd_test.go` 中添加：

```go
func TestFormatEffectiveSDKRows(t *testing.T) {
	rows := []effectiveSDKRow{
		{
			Name:    "go",
			Version: "1.25",
			Source:  sourceDirectory,
			Overrides: []stateValue{
				{Source: sourceSession, Version: "1.24.6"},
				{Source: sourceGlobal, Version: "1.21.13"},
			},
		},
		{
			Name:    "node",
			Version: "22.0.0",
			Source:  sourceSession,
		},
	}

	lines := formatEffectiveSDKRows(rows)

	assert.Eq(t, []string{
		"SDKs:",
		"          go => 1.25  (Directory State)",
		"        node => 22.0.0  (Session Context)",
		"",
		"Overrides:",
		"          go: Session Context 1.24.6, Global State 1.21.13",
	}, lines)
}

func TestFormatSessionContextLines(t *testing.T) {
	session := models.NewActivityState("session.json")
	session.SDKs["go"] = "1.24.6"
	session.SDKs["node"] = "22.0.0"

	lines := formatStateSDKLines("Session Defaults:", session, map[string]string{
		"go": sourceDirectory,
	})

	assert.Contains(t, lines, "Session Defaults:")
	assert.Contains(t, lines, "  <green>        go</> => 1.24.6  (overridden by Directory State)")
	assert.Contains(t, lines, "  <green>      node</> => 22.0.0")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/cli -run "TestFormatEffectiveSDKRows|TestFormatSessionContextLines" -count=1 -v
```

Expected:

```text
FAIL
undefined: formatEffectiveSDKRows
```

- [ ] **Step 3: 实现格式化函数**

在 `internal/cli/status_cmd.go` 中添加：

```go
func formatEffectiveSDKRows(rows []effectiveSDKRow) []string {
	if len(rows) == 0 {
		return []string{"No effective state found"}
	}

	lines := []string{"SDKs:"}
	var overrideLines []string
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%12s => %s  (%s)", row.Name, row.Version, row.Source))
		if len(row.Overrides) == 0 {
			continue
		}
		parts := make([]string, 0, len(row.Overrides))
		for _, item := range row.Overrides {
			parts = append(parts, fmt.Sprintf("%s %s", item.Source, item.Version))
		}
		overrideLines = append(overrideLines, fmt.Sprintf("%12s: %s", row.Name, strings.Join(parts, ", ")))
	}

	if len(overrideLines) > 0 {
		lines = append(lines, "", "Overrides:")
		lines = append(lines, overrideLines...)
	}
	return lines
}

func formatStateSDKLines(title string, state *models.ActivityState, overridden map[string]string) []string {
	if state == nil || len(state.SDKs) == 0 {
		return nil
	}

	lines := []string{title}
	names := make([]string, 0, len(state.SDKs))
	for name := range state.SDKs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		suffix := ""
		if by := overridden[name]; by != "" {
			suffix = fmt.Sprintf("  (overridden by %s)", by)
		}
		lines = append(lines, fmt.Sprintf("  <green>%10s</> => %s%s", name, state.SDKs[name], suffix))
	}
	return lines
}
```

同时添加 import：

```go
import "strings"
```

- [ ] **Step 4: 接入 `StatusCmd`**

将 `StatusCmd` 的 `Func` 改为：

```go
		Func: func(c *gcli.Command, args []string) error {
			return handleStatus(opts)
		},
```

添加：

```go
func handleStatus(opts statusOptions) error {
	if err := xenv.InitState(); err != nil {
		return fmt.Errorf("failed to load activity state: %w", err)
	}

	global := xenv.State().Global()
	session := xenv.State().Session()
	dirStates := xenv.State().DirStates()

	printStatusSection("Effective State", formatEffectiveSDKRows(buildEffectiveSDKRows(global, session, dirStates)))
	if opts.Layers {
		overridden := buildOverriddenSDKSources(global, session, dirStates)
		fmt.Println()
		printStatusSection("Global State", formatLayerLines("Global Defaults:", global, "No global state found", overridden[global]))
		for _, dirState := range dirStates {
			fmt.Println()
			printStatusSection("Directory State", formatLayerLines("Directory SDKs:", dirState, "No directory state found", overridden[dirState]))
		}
		fmt.Println()
		printStatusSection("Session Context", formatLayerLines("Session Defaults:", session, "No session context found", overridden[session]))
	}
	return nil
}
```

所需辅助函数在同文件添加：

```go
func printStatusSection(name string, lines []string) {
	fmt.Printf("[%s]\n", name)
	fmt.Println(strings.Repeat("-", 69))
	for _, line := range lines {
		if strings.HasPrefix(line, "  <green>") {
			ccolor.Println(line)
			continue
		}
		fmt.Println(line)
	}
}

func formatLayerLines(title string, state *models.ActivityState, emptyMessage string, overridden map[string]string) []string {
	if state == nil || state.IsEmpty() {
		return []string{emptyMessage}
	}
	lines := []string{" - from: " + state.File}
	lines = append(lines, formatStateSDKLines(title, state, overridden)...)
	return lines
}
```

添加 imports：

```go
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv"
```

- [ ] **Step 5: 运行测试**

Run:

```bash
go test ./internal/cli -run "TestBuildEffectiveSDKRows|TestFormatEffectiveSDKRows|TestFormatSessionContextLines" -count=1 -v
```

Expected:

```text
PASS
```

- [ ] **Step 6: 提交**

Run:

```bash
git add internal/cli/status_cmd.go internal/cli/status_cmd_test.go
git commit -m "feat: show effective status layers"
```

## Task 4: 移除顶层 `list/ls` 及相关命令代码

**Files:**
- Modify or Delete: `internal/cli/list_cmd.go`
- Modify or Delete: `internal/cli/list_cmd_test.go`
- Modify: `internal/cli/app_test.go`
- Test: `go test ./internal/cli -count=1`

- [ ] **Step 1: 删除顶层 list 命令代码**

如果 `internal/cli/list_cmd.go` 中只有顶层 `list` 相关逻辑，删除文件：

```bash
git rm internal/cli/list_cmd.go internal/cli/list_cmd_test.go
```

如果其中有仍被其他命令使用的 helper，先移动到对应文件：

- `listSDKs()` 移动到 `internal/cli/sdk_cmd.go` 或直接删除，因为 `SDKListCmd` 已经直接调用 `sdkSvc.ListSDKs(opts.All)`。
- `listEnvs()` 保留在 `internal/cli/env_cmd.go`。
- `listEnvPaths()` 保留在 `internal/cli/path_cmd.go`。
- 状态展示 helper 由 `internal/cli/status_cmd.go` 承担。

- [ ] **Step 2: 运行测试**

Run:

```bash
go test ./internal/cli -count=1
```

Expected:

```text
PASS
```

- [ ] **Step 3: 手动确认 help**

Run:

```bash
go run ./cmd/xenv --help
```

Expected:

```text
Available Commands:
  check
  config
  env
  path
  sdk
  shell
  status
  unuse
  use
```

The output must not include top-level `list`.

- [ ] **Step 4: 提交**

Run:

```bash
git add internal/cli
git commit -m "refactor: remove top-level list command"
```

## Task 5: 更新 shell completion

**Files:**
- Modify: `internal/xenv/shell/gen_hook_bash.go`
- Modify: `internal/xenv/shell/gen_hook_zsh.go`
- Modify: `internal/xenv/shell/gen_hook_pwsh.go`
- Modify: `internal/xenv/shell/gen_hook_test.go`
- Test: `go test ./internal/xenv/shell -count=1`

- [ ] **Step 1: 更新测试断言**

在 `internal/xenv/shell/gen_hook_test.go` 的 command aliases 测试中，把 completion 断言改为包含 `status st` 且不包含 `list`：

```go
assertContains(t, bash, `complete -W "use u unuse un env e set unset path p status st help" xenv`)
assertNotContains(t, bash, ` path p list help`)

assertContains(t, zsh, `compctl -k "use u unuse un env e set unset path p status st help" xenv`)
assertNotContains(t, zsh, ` path p list help`)

assertContains(t, pwsh, "@('use', 'u', 'unuse', 'un', 'env', 'e', 'set', 'unset', 'path', 'p', 'status', 'st', '--help')")
assertNotContains(t, pwsh, "'list'")
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/xenv/shell -run TestGeneratedHooksEvaluateCommandAliases -count=1 -v
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 更新 hook 模板**

在 `internal/xenv/shell/gen_hook_bash.go` 中替换 completion：

```bash
complete -W "use u unuse un env e set unset path p status st help" xenv
```

在 `internal/xenv/shell/gen_hook_zsh.go` 中替换 completion：

```bash
compctl -k "use u unuse un env e set unset path p status st help" xenv
```

在 `internal/xenv/shell/gen_hook_pwsh.go` 中替换 completion array：

```powershell
@('use', 'u', 'unuse', 'un', 'env', 'e', 'set', 'unset', 'path', 'p', 'status', 'st', '--help') | Where-Object { $_ -like "$wordToComplete*" }
```

- [ ] **Step 4: 运行 shell 测试**

Run:

```bash
go test ./internal/xenv/shell -count=1
```

Expected:

```text
PASS
```

- [ ] **Step 5: 提交**

Run:

```bash
git add internal/xenv/shell
git commit -m "fix: update hook completions for status"
```

## Task 6: 目录状态冲突时提示临时 runtime override

**Files:**
- Modify: `internal/xenv/service/sdk_service.go`
- Modify: `internal/xenv/service/tool_service_test.go`
- Test: `go test ./internal/xenv/service -run TestActivateSDKWarnsWhenSessionUseOverridesDirectoryState -count=1 -v`

- [ ] **Step 1: 写失败测试**

在 `internal/xenv/service/tool_service_test.go` 中添加：

```go
func TestActivateSDKWarnsWhenSessionUseOverridesDirectoryState(t *testing.T) {
	_, _, svc, _ := newDirenvTestService(t, "test-session-override-warning", func(projectDir string) {
		xenvToml := filepath.Join(projectDir, ".xenv.toml")
		err := os.WriteFile(xenvToml, []byte("[sdks]\n  go = \"1.23\"\n"), 0o644)
		assert.Require(t, assert.NoErr(t, err))
	})

	out := captureColorOutput(t, func() {
		_, err := svc.ActivateSDKs([]string{"go:1.24"}, models.OpFlagSession)
		assert.Require(t, assert.NoErr(t, err))
	})

	assert.Contains(t, out, "Activate go:1.24")
	assert.Contains(t, out, "WARN: directory state wants go:1.23")
	assert.Contains(t, out, "temporary runtime override")
	assert.Contains(t, out, "xenv use -s go:1.24")
}
```

如果当前测试文件没有 `captureColorOutput`，新增：

```go
func captureColorOutput(t *testing.T, fn func()) string {
	t.Helper()
	buf := byteutil.NewBuffer()
	color.SetOutput(buf)
	t.Cleanup(color.ResetOutput)
	fn()
	return buf.String()
}
```

需要 imports：

```go
	"github.com/gookit/color"
	"github.com/gookit/goutil/byteutil"
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/xenv/service -run TestActivateSDKWarnsWhenSessionUseOverridesDirectoryState -count=1 -v
```

Expected:

```text
FAIL
Should contain: "temporary runtime override"
```

- [ ] **Step 3: 实现 warning**

在 `internal/xenv/service/sdk_service.go` 的 `activateSDKs` 中，成功匹配 `localSDK` 后、打印 activate message 后，调用：

```go
ts.warnTemporaryRuntimeOverride(spec, opFlag)
```

新增方法：

```go
func (ts *SDKService) warnTemporaryRuntimeOverride(spec *models.VersionSpec, opFlag models.OpFlag) {
	if opFlag != models.OpFlagSession {
		return
	}
	deState := ts.state.Nearest()
	if deState == nil || deState.IsEmpty() {
		return
	}
	dirVersion := deState.SDKs[spec.Name]
	if dirVersion == "" || dirVersion == spec.Version || dirVersion == spec.RealVersion {
		return
	}

	ccolor.Warnf(
		"WARN: directory state wants %s:%s; this activation is a temporary runtime override. Use `xenv use -s %s:%s` to update .xenv.toml.\n",
		spec.Name,
		dirVersion,
		spec.Name,
		spec.Version,
	)
}
```

- [ ] **Step 4: 运行测试**

Run:

```bash
go test ./internal/xenv/service -run TestActivateSDKWarnsWhenSessionUseOverridesDirectoryState -count=1 -v
```

Expected:

```text
PASS
```

- [ ] **Step 5: 提交**

Run:

```bash
git add internal/xenv/service/sdk_service.go internal/xenv/service/tool_service_test.go
git commit -m "feat: warn on temporary session sdk override"
```

## Task 7: 更新 `check` 文案和状态语义文档

**Files:**
- Modify: `internal/cli/check_cmd.go`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/design/2026-06-12-xenv-state-semantics-design.md`
- Test: `go test ./internal/cli -count=1`

- [ ] **Step 1: 更新 `check` 文案**

修改 `internal/cli/check_cmd.go`：

```go
var CheckCmd = &gcli.Command{
	Name: "check",
	Desc: "Check effective SDKs and project tool requirements",
```

```go
func CheckSDKCmd() *gcli.Command {
	return &gcli.Command{
		Name: "sdk",
		Desc: "Check SDK availability for effective state",
```

- [ ] **Step 2: 更新 README 命令结构**

在 `README.zh-CN.md` 中加入或更新命令结构说明：

````markdown
### 命令职责

- `xenv status`: 查看当前目录和当前 shell 的状态，包括 Effective State、Session Context 和 Runtime State。
- `xenv sdk list`: 列出本机 SDK inventory。
- `xenv env list`: 列出 xenv 管理的 ENV 状态。
- `xenv path list`: 列出 xenv 管理的 PATH 状态。
- `xenv check`: 校验 Effective State 和项目 tool requirements 是否满足。

顶层 `xenv list` / `xenv ls` 在 v0 中不保留；状态诊断请使用 `xenv status`。
````

在 `README.md` 中加入英文对应说明：

````markdown
### Command Responsibilities

- `xenv status`: Show current state for this directory and shell, including Effective State, Session Context, and Runtime State.
- `xenv sdk list`: List local SDK inventory.
- `xenv env list`: List xenv-managed ENV state.
- `xenv path list`: List xenv-managed PATH state.
- `xenv check`: Check whether Effective State and project tool requirements are satisfied.

Top-level `xenv list` / `xenv ls` is not kept in v0; use `xenv status` for state diagnostics.
````

- [ ] **Step 3: 更新设计文档**

确认 `docs/design/2026-06-12-xenv-state-semantics-design.md` 包含：

```text
xenv status
xenv status --layers
xenv status --runtime
```

并确认文档没有再推荐：

```text
xenv ls --group
xenv ls --group --runtime
```

- [ ] **Step 4: 运行测试**

Run:

```bash
go test ./internal/cli -count=1
```

Expected:

```text
PASS
```

- [ ] **Step 5: 提交**

Run:

```bash
git add internal/cli/check_cmd.go README.md README.zh-CN.md docs/design/2026-06-12-xenv-state-semantics-design.md
git commit -m "docs: document status command semantics"
```

## Task 8: Runtime State 检测设计占位

**Files:**
- Create: `docs/design/xenv-runtime-state-detection.md`
- Test: no code test

- [ ] **Step 1: 创建 Runtime 检测设计文档**

创建 `docs/design/xenv-runtime-state-detection.md`：

````markdown
# xenv Runtime State Detection Design

## 修订记录

| 日期 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-13 | v0.1 | Codex | 初版，定义 `xenv status --runtime` 的 PATH-based runtime 检测方案。 |

## 关联文档

- 状态语义设计: [2026-06-12-xenv-state-semantics-design.md](2026-06-12-xenv-state-semantics-design.md)
- 实施计划: [../plans/2026-06-12-effective-state-display-implementation.md](../plans/2026-06-12-effective-state-display-implementation.md)

## 目标

`xenv status --runtime` 从当前进程环境检测 Runtime State，并与 Effective State 对比，输出 mismatch warning。

## 第一版范围

1. 只检测 xenv 已知 SDK 的 bin 目录。
2. 不解析 `go version`、`java -version` 等命令输出。
3. 不修改 shell 环境。
4. 不自动修复 PATH。

## 检测流程

1. 计算 Effective State。
2. 根据 SDK index 找到 Effective SDK 对应 bin dir。
3. 读取当前 `PATH`。
4. 扫描 PATH 中第一个匹配同名 SDK 的 bin dir。
5. 若 runtime bin dir 与 effective bin dir 不一致，输出 warning。

## 输出示例

```text
[Runtime State]
---------------------------------------------------------------------
go:
  actual:   1.24.6
  bin:      D:\work\env\devsdk\gosdk\go1.24.6\bin
  expected: 1.23 from Directory State
  expected bin:
            D:\work\env\devsdk\gosdk\go1.23.12\bin
```
````

- [ ] **Step 2: 提交**

Run:

```bash
git add docs/design/xenv-runtime-state-detection.md
git commit -m "docs: design runtime state detection"
```

## 验收标准

完成第一阶段后：

1. 顶层 `xenv status` 存在，`xenv list` / `xenv ls` 不再注册。
2. `xenv status` 展示 `[Effective State]`。
3. `xenv status --layers` 展示 `[Global State]`、`[Directory State]`、`[Session Context]`。
4. Session 分组内部使用 `Session Defaults`，不使用容易误解的 `Active Develop SDKs`。
5. Directory State 覆盖 Session/Global 时，低优先级项显示 `(overridden by Directory State)`。
6. 在有 `.xenv.toml` 且同名 SDK 冲突的目录中执行普通 `xenv use`，输出 temporary runtime override warning。
7. hook completion 包含 `status` / `st`，不包含顶层 `list`。
8. `go test ./...` 通过。

## 风险与注意事项

- 展示层不要调用 `ActivityState.Merge()` 计算 Effective State，因为它会设置 `HasUpdate`，可能导致无意保存状态。
- Runtime State 不能从 session JSON 推导；session JSON 只表示 Session Context。
- 移除顶层 `list` 后，README、completion、app 注册测试必须同步更新。
- `xenv check` 的行为可以先不重写，但文案要避免 “active SDKs” 这种含混表达。
- Runtime PATH 检测作为第二阶段，不要和第一阶段状态展示混在同一个提交里。

## 实施顺序

1. Task 1: 注册 `status`，移除顶层 `list` 注册。
2. Task 2: 写 Effective State 纯计算。
3. Task 3: 实现 `status` 输出和 `--layers`。
4. Task 4: 删除或收缩旧 `list` 代码。
5. Task 5: 更新 hook completion。
6. Task 6: 加 session override warning。
7. Task 7: 更新 check 文案和 README。
8. Task 8: 新增 Runtime 检测设计文档。

每个任务单独提交。涉及 Go 代码后至少运行对应包测试；全部完成后运行：

```bash
go test ./...
```
