# xenv Effective State Display Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 明确 `xenv` 的 global/session/directory/runtime 状态语义，并优化 `xenv ls --group`，让用户能看出当前目录下哪个 SDK 最终生效、哪个状态被覆盖。

**Architecture:** 第一阶段只改展示层，不改变 `use`、`unuse`、`init-direnv` 的激活行为和状态写入行为。`xenv ls --group` 保留原始分组状态，同时新增 `Effective State`，按 `Directory State > Session State > Global State` 计算展示，并标注被覆盖的来源。第二阶段再评估 runtime 环境检测，用于提示当前 shell PATH/ENV 是否尚未执行 `init-direnv` 或已和状态文件脱节。

**Tech Stack:** Go, `gookit/gcli/v3`, `github.com/gookit/goutil/x/assert`, TOML/JSON state files, PowerShell/bash/zsh hook runtime environment.

---

## 修订记录

| 日期 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-12 | v0.1 | Codex | 初版实施计划，聚焦 Effective State 展示和后续 Runtime 检测评估。 |
| 2026-06-12 | v0.2 | Codex | 补充状态语义设计文档链接，明确 Session State 是 Session Defaults/Context，不是 Runtime State。 |

## 关联文档

- 设计文档: [../design/2026-06-12-xenv-state-semantics-design.md](../design/2026-06-12-xenv-state-semantics-design.md)

## 背景

当前 `xenv ls --group` 展示的是三个持久状态源：

- `Global State`: `~/.config/xenv/global.toml`
- `Directory States`: 当前目录或父目录的 `.xenv.local.toml` / `.xenv.toml`
- `Session State`: `~/.config/xenv/session/<id>.json`

但 `go version` 等命令读取的是当前 shell 的实际 `PATH` / `GOROOT` / `JAVA_HOME` 等环境变量。用户在同一个目录里可能看到：

```text
Directory States:
          go => 1.25

Session State:
          go => 1.24.6

go version:
          go1.25.7
```

这不是单纯状态读取错误，而是展示语义不清晰：`Session State` 是当前 shell 的可恢复默认状态，不一定等于当前目录下实际应该生效的状态；项目目录的 `.xenv.toml` 应该优先表达项目约束。

## 目标语义

实施时按以下语义处理：

```text
Directory State > Session State > Global State
```

定义：

- `Global State`: 机器级默认值，只在没有更近约束时生效。
- `Session State`: 当前 shell 的持久化上下文。展示层应把其中的 SDK/ENV/PATH 称为 `Session Defaults`，用于无目录约束的目录，或离开目录后恢复；它不是 Runtime State。
- `Directory State`: 项目级约束，进入目录后优先于 global 和普通 session。
- `Effective State`: 根据上面的优先级推导出来的“当前目录期望生效状态”。
- `Runtime State`: 当前 shell 真实环境，来自 hook eval 执行结果，可能与 Effective State 暂时不一致；后续应从当前 PATH/ENV 检测，而不是从 session JSON 直接得出。

第一阶段不实现离开目录恢复，也不把 direnv 激活结果写回 `session.SDKs`。这样可以保留 session 作为恢复来源，避免进入项目目录后污染用户原本的 session 默认值。

## 非目标

- 不修改 `.xenv.toml` 格式。
- 不修改 `xenv use/unuse` 的持久化语义。
- 不修改 `init-direnv` 激活脚本生成逻辑。
- 不在第一阶段解析真实 `go version` 输出。
- 不因为目录状态覆盖 session 就删除 session state 中的 SDK 记录。

## 文件结构

计划修改或新增这些文件：

- `internal/cli/list_cmd.go`: 新增 Effective State 展示、SDK 来源计算、overridden 标注。
- `internal/cli/list_cmd_test.go`: 增加 Effective State 格式化单元测试，覆盖 global/session/direnv 覆盖关系。
- `internal/xenv/manager/state_manager_test.go`: 如需要，补充 direnv 查找顺序测试；第一阶段优先不修改 manager。
- `docs/plans/2026-06-12-effective-state-display-implementation.md`: 本实施计划。

后续第二阶段如果实现 runtime 检测，预计新增：

- `internal/cli/runtime_state.go`: 从当前进程环境推导 PATH/ENV 中可能生效的 SDK 线索。
- `internal/cli/runtime_state_test.go`: 覆盖 PATH 中 SDK bin 路径识别、mismatch warning。

## 展示设计

当前 `xenv ls --group` 建议调整为：

```text
[Effective State]
---------------------------------------------------------------------
Active Develop SDKs:
          go => 1.25  (Directory State)

Overrides:
          go: Session State 1.24.6, Global State 1.21.13

[Global State]
---------------------------------------------------------------------
 - from: C:\Users\inhere/.config/xenv/global.toml
Active Develop SDKs:
          go => 1.21.13  (overridden by Directory State)

[Directory States]
---------------------------------------------------------------------
 - from: D:\work\aidev\lite-tools\xenv\.xenv.toml
Active Develop SDKs:
          go => 1.25

[Session State]
---------------------------------------------------------------------
 - from: C:\Users\inhere/.config/xenv/session/xenv_6c63a33695.json
Active Develop SDKs:
          go => 1.24.6  (overridden by Directory State)
```

如果某类状态为空，仍保持现有行为：

```text
[Global State]
---------------------------------------------------------------------
 - from: ...
No global state found
```

如果没有覆盖关系，`Overrides` 段不显示。

## Task 1: 为 Effective SDK 计算写失败测试

**Files:**
- Modify: `internal/cli/list_cmd_test.go`
- Test: `go test ./internal/cli -run TestBuildEffectiveSDKRows -count=1 -v`

- [ ] **Step 1: 增加测试用例**

在 `internal/cli/list_cmd_test.go` 中添加：

```go
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
	assert.Eq(t, "go", rows[0].Name)
	assert.Eq(t, "1.25", rows[0].Version)
	assert.Eq(t, "Directory State", rows[0].Source)
	assert.Eq(t, []stateValue{
		{Source: "Session State", Version: "1.24.6"},
		{Source: "Global State", Version: "1.21.13"},
	}, rows[0].Overrides)

	assert.Eq(t, "java", rows[1].Name)
	assert.Eq(t, "17.0.11", rows[1].Version)
	assert.Eq(t, "Global State", rows[1].Source)

	assert.Eq(t, "node", rows[2].Name)
	assert.Eq(t, "22.0.0", rows[2].Version)
	assert.Eq(t, "Session State", rows[2].Source)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/cli -run TestBuildEffectiveSDKRows -count=1 -v
```

Expected:

```text
FAIL: undefined: buildEffectiveSDKRows
```

## Task 2: 实现 Effective SDK 计算

**Files:**
- Modify: `internal/cli/list_cmd.go`
- Test: `go test ./internal/cli -run TestBuildEffectiveSDKRows -count=1 -v`

- [ ] **Step 1: 新增内部结构**

在 `internal/cli/list_cmd.go` 中加入：

```go
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

- [ ] **Step 2: 新增纯计算函数**

在 `internal/cli/list_cmd.go` 中加入：

```go
func buildEffectiveSDKRows(global, session *models.ActivityState, dirStates []*models.ActivityState) []effectiveSDKRow {
	values := make(map[string][]stateValue)

	collectSDKValues(values, "Global State", global)
	collectSDKValues(values, "Session State", session)
	for _, dirState := range dirStates {
		collectSDKValues(values, "Directory State", dirState)
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

同时在 import 中添加：

```go
import "sort"
```

- [ ] **Step 3: 运行测试确认通过**

Run:

```bash
go test ./internal/cli -run TestBuildEffectiveSDKRows -count=1 -v
```

Expected:

```text
PASS
```

- [ ] **Step 4: 提交**

Run:

```bash
git add internal/cli/list_cmd.go internal/cli/list_cmd_test.go
git commit -m "test: cover effective sdk state rows"
```

## Task 3: 在 `ls --group` 中展示 Effective State

**Files:**
- Modify: `internal/cli/list_cmd.go`
- Modify: `internal/cli/list_cmd_test.go`
- Test: `go test ./internal/cli -run "TestFormatEffectiveSDKRows|TestListStateGroup" -count=1 -v`

- [ ] **Step 1: 增加格式化测试**

在 `internal/cli/list_cmd_test.go` 中添加：

```go
func TestFormatEffectiveSDKRows(t *testing.T) {
	rows := []effectiveSDKRow{
		{
			Name:    "go",
			Version: "1.25",
			Source:  "Directory State",
			Overrides: []stateValue{
				{Source: "Session State", Version: "1.24.6"},
				{Source: "Global State", Version: "1.21.13"},
			},
		},
		{
			Name:    "node",
			Version: "22.0.0",
			Source:  "Session State",
		},
	}

	lines := formatEffectiveSDKRows(rows)

	assert.Eq(t, []string{
		"Active Develop SDKs:",
		"          go => 1.25  (Directory State)",
		"        node => 22.0.0  (Session State)",
		"",
		"Overrides:",
		"          go: Session State 1.24.6, Global State 1.21.13",
	}, lines)
}
```

- [ ] **Step 2: 实现 `formatEffectiveSDKRows`**

在 `internal/cli/list_cmd.go` 中加入：

```go
func formatEffectiveSDKRows(rows []effectiveSDKRow) []string {
	if len(rows) == 0 {
		return []string{"No effective state found"}
	}

	lines := []string{"Active Develop SDKs:"}
	var overrideLines []string
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  <green>%10s</> => %s  (%s)", row.Name, row.Version, row.Source))
		if len(row.Overrides) == 0 {
			continue
		}
		parts := make([]string, 0, len(row.Overrides))
		for _, item := range row.Overrides {
			parts = append(parts, fmt.Sprintf("%s %s", item.Source, item.Version))
		}
		overrideLines = append(overrideLines, fmt.Sprintf("  <green>%10s</>: %s", row.Name, strings.Join(parts, ", ")))
	}

	if len(overrideLines) > 0 {
		lines = append(lines, "", "Overrides:")
		lines = append(lines, overrideLines...)
	}
	return lines
}
```

- [ ] **Step 3: 接入 `handleListState`**

将 `handleListState(true)` 的开头改为先展示 Effective State：

```go
	global := xenv.State().Global()
	dirStates := xenv.State().DirStates()
	session := xenv.State().Session()

	listEffectiveStateGroup(global, session, dirStates)

	fmt.Println()
	listStateGroup("Global State", global, "No global state found")
```

新增：

```go
func listEffectiveStateGroup(global, session *models.ActivityState, dirStates []*models.ActivityState) {
	tl := title.New("", func(t *title.Title) {
		t.Color = "ylw1"
		t.PercentWidth = 80
		t.PaddingLR = false
		t.ShowBorder = true
	})
	tl.ShowNew("[Effective State]")
	for _, line := range formatEffectiveSDKRows(buildEffectiveSDKRows(global, session, dirStates)) {
		if line == "" {
			fmt.Println()
			continue
		}
		if strings.HasPrefix(line, "  ") {
			ccolor.Println(line)
			continue
		}
		ccolor.Cyanln(line)
	}
}
```

- [ ] **Step 4: 运行测试**

Run:

```bash
go test ./internal/cli -run "TestBuildEffectiveSDKRows|TestFormatEffectiveSDKRows" -count=1 -v
```

Expected:

```text
PASS
```

- [ ] **Step 5: 手动查看输出**

Run:

```bash
go run ./cmd/xenv ls --group
```

Expected:

```text
[Effective State]
...
[Global State]
...
```

- [ ] **Step 6: 提交**

Run:

```bash
git add internal/cli/list_cmd.go internal/cli/list_cmd_test.go
git commit -m "feat: show effective state in grouped list"
```

## Task 4: 标注原始分组中的 overridden SDK

**Files:**
- Modify: `internal/cli/list_cmd.go`
- Modify: `internal/cli/list_cmd_test.go`
- Test: `go test ./internal/cli -run TestFormatActivityLinesWithOverrides -count=1 -v`

- [ ] **Step 1: 增加测试**

在 `internal/cli/list_cmd_test.go` 中添加：

```go
func TestFormatActivityLinesWithOverrides(t *testing.T) {
	state := models.NewActivityState("session.json")
	state.SDKs["go"] = "1.24.6"
	state.SDKs["node"] = "22.0.0"

	overridden := map[string]string{
		"go": "Directory State",
	}

	lines := formatActivityLinesWithOverrides(state, overridden)

	assert.Contains(t, lines, "  <green>        go</> => 1.24.6  (overridden by Directory State)")
	assert.Contains(t, lines, "  <green>      node</> => 22.0.0")
}
```

- [ ] **Step 2: 实现 override-aware 格式化**

保留现有 `formatActivityLines(state)`，新增：

```go
func formatActivityLinesWithOverrides(state *models.ActivityState, overridden map[string]string) []string {
	var lines []string
	if len(state.SDKs) > 0 {
		lines = append(lines, "Active Develop SDKs:")
		names := make([]string, 0, len(state.SDKs))
		for name := range state.SDKs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			version := state.SDKs[name]
			suffix := ""
			if by := overridden[name]; by != "" {
				suffix = fmt.Sprintf("  (overridden by %s)", by)
			}
			lines = append(lines, fmt.Sprintf("  <green>%10s</> => %s%s", name, version, suffix))
		}
	}

	lines = appendEnvPathToolLines(lines, state)
	return lines
}
```

为避免复制 ENV/PATH/Tools 格式化逻辑，将现有 `formatActivityLines` 中 SDK 以外的部分提取为：

```go
func appendEnvPathToolLines(lines []string, state *models.ActivityState) []string {
	if len(state.Envs) > 0 {
		lines = appendActivitySection(lines, "Active Env Variables:")
		for name, value := range state.Envs {
			lines = append(lines, fmt.Sprintf("  <green>%s</>=%s", name, value))
		}
	}

	if len(state.Paths) > 0 {
		lines = appendActivitySection(lines, "Active PATH Entries:")
		for i, path := range state.Paths {
			lines = append(lines, fmt.Sprintf("  <green>%d</>. %s", i+1, path))
		}
	}

	if len(state.ToolRequirements) > 0 {
		lines = appendActivitySection(lines, "Tool Requirements:")
		for name, requirement := range state.ToolRequirements {
			lines = append(lines, fmt.Sprintf("  <green>%s</> => %s", name, requirement))
		}
	}
	return lines
}
```

- [ ] **Step 3: 为各分组计算 overridden map**

新增函数：

```go
func buildOverriddenSDKSources(global, session *models.ActivityState, dirStates []*models.ActivityState) map[*models.ActivityState]map[string]string {
	result := make(map[*models.ActivityState]map[string]string)
	rows := buildEffectiveSDKRows(global, session, dirStates)

	for _, row := range rows {
		for _, item := range row.Overrides {
			var target *models.ActivityState
			switch item.Source {
			case "Global State":
				target = global
			case "Session State":
				target = session
			case "Directory State":
				for _, dirState := range dirStates {
					if dirState.SDKs[row.Name] == item.Version {
						target = dirState
						break
					}
				}
			}
			if target == nil {
				continue
			}
			if result[target] == nil {
				result[target] = make(map[string]string)
			}
			result[target][row.Name] = row.Source
		}
	}
	return result
}
```

- [ ] **Step 4: 调整 `listStateGroup` 支持 overridden**

将签名改为：

```go
func listStateGroup(titleText string, state *models.ActivityState, emptyMessage string, overridden map[string]string)
```

内部调用：

```go
listActivityLines(formatActivityLinesWithOverrides(state, overridden))
```

同时新增：

```go
func listActivityLines(lines []string) {
	for _, line := range lines {
		if line == "" {
			fmt.Println()
			continue
		}
		if strings.HasPrefix(line, "  ") {
			ccolor.Println(line)
			continue
		}
		ccolor.Cyanln(line)
	}
}
```

让 `listActivity(state)` 复用：

```go
func listActivity(state *models.ActivityState) {
	listActivityLines(formatActivityLines(state))
}
```

- [ ] **Step 5: 更新调用点**

在 `handleListState(true)` 中：

```go
overridden := buildOverriddenSDKSources(global, session, dirStates)

listStateGroup("Global State", global, "No global state found", overridden[global])
...
listStateGroup("", dirState, "No directory state found", overridden[dirState])
...
listStateGroup("Session State", sess, "No session state found", overridden[sess])
```

- [ ] **Step 6: 运行测试**

Run:

```bash
go test ./internal/cli -count=1
```

Expected:

```text
PASS
```

- [ ] **Step 7: 提交**

Run:

```bash
git add internal/cli/list_cmd.go internal/cli/list_cmd_test.go
git commit -m "feat: mark overridden grouped state entries"
```

## Task 5: 文档化状态语义

**Files:**
- Modify: `README.zh-CN.md`
- Modify: `README.md`
- Test: `go test ./...`

- [ ] **Step 1: 在中文 README 添加状态语义**

在 `README.zh-CN.md` 的 shell hook 或 state/list 相关章节加入：

````markdown
### 状态优先级

`xenv` 会同时维护多类状态：

- Global State: 用户全局默认值，保存到 `~/.config/xenv/global.toml`
- Session State: 当前 shell 的临时默认值，保存到 `~/.config/xenv/session/<id>.json`
- Directory State: 当前项目目录配置，保存到 `.xenv.toml` 或 `.xenv.local.toml`
- Effective State: 当前目录下按优先级推导出的期望生效状态

优先级为：

```text
Directory State > Session State > Global State
```

因此，如果当前 shell session 中有 `go => 1.24.6`，但项目 `.xenv.toml` 中配置了 `go => 1.25`，进入该目录后目录配置会优先生效。`xenv ls --group` 会显示原始分组状态，并在 `Effective State` 中展示当前目录期望生效的结果。
````

- [ ] **Step 2: 在英文 README 添加同等说明**

在 `README.md` 对应章节加入：

````markdown
### State Priority

`xenv` keeps multiple state layers:

- Global State: user-wide defaults in `~/.config/xenv/global.toml`
- Session State: temporary defaults for the current shell in `~/.config/xenv/session/<id>.json`
- Directory State: project state in `.xenv.toml` or `.xenv.local.toml`
- Effective State: the expected state for the current directory after applying priority

Priority:

```text
Directory State > Session State > Global State
```

If the current shell session contains `go => 1.24.6` and the project `.xenv.toml` contains `go => 1.25`, the directory state wins after the direnv hook is applied. `xenv ls --group` shows both the raw state groups and the effective state for the current directory.
````

- [ ] **Step 3: 运行测试**

Run:

```bash
go test ./...
```

Expected:

```text
PASS
```

- [ ] **Step 4: 提交**

Run:

```bash
git add README.md README.zh-CN.md
git commit -m "docs: explain xenv state priority"
```

## Task 6: 第二阶段调研 Runtime Environment 检测

**Files:**
- Create: `docs/design/xenv-runtime-state-detection.md`
- No code change in this task.

- [ ] **Step 1: 创建设计文档**

创建 `docs/design/xenv-runtime-state-detection.md`：

```markdown
# xenv Runtime State Detection Design

## 修订记录

| 日期 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-12 | v0.1 | Codex | 初版，评估 runtime 环境检测方案。 |

## 关联文档

- 实施计划: [../plans/2026-06-12-effective-state-display-implementation.md](../plans/2026-06-12-effective-state-display-implementation.md)

## 目标

检测当前 shell runtime 环境是否与 `Effective State` 一致，用于提示用户是否需要执行 `cd .`、`xenv init-direnv` 或重新打开 shell。

## 初步方案

第一版只检测 PATH 中 SDK bin 目录：

1. 根据 Effective State 找到对应 SDK 的 install dir 和 bin dir。
2. 读取当前进程 `PATH`。
3. 判断期望 SDK bin 是否存在且排在同名 SDK bin 之前。
4. 如果不一致，在 `xenv ls --group` 增加 `Runtime Warning`。

## 不处理

- 不解析任意第三方 SDK 的版本命令输出。
- 不修改 shell 环境。
- 不自动修复 PATH。
```

- [ ] **Step 2: 提交设计文档**

Run:

```bash
git add docs/design/xenv-runtime-state-detection.md docs/plans/2026-06-12-effective-state-display-implementation.md
git commit -m "docs: plan runtime state detection"
```

## 验收标准

完成第一阶段后，以下场景应成立：

1. `xenv ls --group` 总是先显示 `[Effective State]`。
2. 如果 `.xenv.toml` 和 session 同时配置同一个 SDK，Effective State 使用 directory 值。
3. 被覆盖的 session/global SDK 会在原始分组里显示 `(overridden by Directory State)`。
4. 空分组仍保持现有 `No ... state found` 行为。
5. 不改变 `xenv use`、`xenv unuse`、`xenv init-direnv` 的状态写入和激活行为。
6. `go test ./...` 通过。

## 风险与注意事项

- `ActivityState.Merge()` 会设置 `HasUpdate`，展示层不要用它计算 Effective State，否则可能污染状态并触发保存。
- map 遍历顺序不稳定，新增格式化测试必须排序 SDK 名称。
- `Directory State > Session State > Global State` 是展示和产品语义目标，但不要在第一阶段重排底层 `LoadStateFiles()` 的 merge 顺序，避免影响激活逻辑。
- Runtime Environment 检测单独做第二阶段，避免把 PATH 识别、SDK 版本解析和展示语义混在一个提交里。

## 实施顺序

1. Task 1 + Task 2: 先完成纯函数 Effective SDK 计算。
2. Task 3: 接入 `ls --group`，用户可以看到 Effective State。
3. Task 4: 标注原始分组中的 overridden。
4. Task 5: 更新 README 状态语义。
5. Task 6: 输出 runtime 检测设计，不急于实现。

每个任务完成后单独提交。涉及 Go 代码后至少运行对应包测试；第一阶段全部完成后运行：

```bash
go test ./...
```
