# xenv 状态语义设计

## 修订记录

| 日期 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-12 | v0.1 | Codex | 初版，明确 Global、Directory、Session、Effective、Runtime State 的职责、触发边界和展示语义。 |
| 2026-06-13 | v0.2 | Codex | 确认 v0 不保留 `list/ls` 兼容入口，状态诊断收敛到一等 `status` 命令，`Session State` 展示为 `Session Context`。 |

## 关联文档

- 实施计划: [../plans/2026-06-12-effective-state-display-implementation.md](../plans/2026-06-12-effective-state-display-implementation.md)

## 背景

`xenv` 当前同时维护多类状态：

- 用户全局状态: `~/.config/xenv/global.toml`
- 项目目录状态: `.xenv.toml` / `.xenv.local.toml`
- shell 会话状态: `~/.config/xenv/session/<session-id>.json`
- 当前 shell 真实环境: `PATH`、`GOROOT`、`JAVA_HOME` 等环境变量

这些状态在大多数简单场景中看起来是一致的。例如执行：

```bash
xenv use go:1.24
go version
```

通常会看到 session JSON 和 `go version` 都指向 `go1.24.x`。

但进入含有 `.xenv.toml` 的项目目录后，会出现合理但容易误解的差异：

```text
[Directory States]
          go => 1.25

[Session State]
          go => 1.24.6

go version
          go1.25.7
```

这个差异不是单纯 bug。它说明“持久状态展示”和“当前 shell 实际环境”是两个层面。设计上必须把以下概念拆开：

- `Session State` 是 xenv 的当前 shell 会话上下文和默认偏好，不是 runtime 真相。
- `Effective State` 是按配置优先级推导出来的当前目录期望状态。
- `Runtime State` 是当前 shell 中真实生效的 `PATH` / ENV，来自 shell hook eval 后的结果。

## 设计目标

1. 明确每类状态的职责和边界，避免 `Session State` 被误解为 runtime 真相。
2. 保留 session JSON 作为 xenv 跨进程的 shell 会话记忆。
3. 明确 `Directory State > Session State > Global State` 是 Effective State 的推导优先级。
4. 明确 Runtime State 由 shell 执行表达式触发，不能只依赖持久化文件判断。
5. 为 `xenv status` 后续展示 Effective State、Session Context、Runtime State 和 Runtime Warning 提供语义基础。

## 非目标

1. 本文档不要求立即修改 `.xenv.toml` 格式。
2. 本文档不要求立即迁移 session JSON 结构。
3. 本文档不要求把 Runtime State 持久化为事实来源。
4. 本文档不要求改变 `xenv use`、`xenv use -s`、`xenv use -g` 的基本命令入口。
5. 本文档不解决“离开目录自动恢复 session default”的完整生命周期实现，只定义该能力需要的状态边界。

## 核心概念

### Global State

Global State 是用户级默认配置。

存储位置：

```text
~/.config/xenv/global.toml
```

职责：

- 表示跨 shell、跨目录的默认 SDK / ENV / PATH。
- 在没有更高优先级状态时作为默认来源。
- 由 `xenv use -g`、`xenv unuse -g`、`xenv set -g`、`xenv path add -g` 等命令修改。

不负责：

- 不表示当前 shell 一定实际使用了什么。
- 不覆盖项目目录的 `.xenv.toml`。

### Directory State

Directory State 是项目级约束。

存储位置：

```text
.xenv.local.toml
.xenv.toml
```

查找规则：

- 从当前目录向父目录查找。
- 同一目录下 `.xenv.local.toml` 优先于 `.xenv.toml`。
- 当前实现只加载最近的一个目录状态文件。

职责：

- 表示项目期望使用的 SDK / ENV / PATH。
- 进入目录或执行 `xenv init-direnv` 时，由 shell hook 应用到 Runtime State。
- 在 Effective State 中优先于 Session State 和 Global State。
- 由 `xenv use -s`、`xenv unuse -s`、`xenv set -s`、`xenv path add -s` 等命令修改。

不负责：

- 不应该污染 session defaults。
- 不应该因为被应用到 Runtime State 就写入 `session.SDKs`。

### Session State

Session State 是当前 shell 会话的 xenv 上下文。为了避免误解，可以在展示层把其中的 SDK / ENV / PATH 称为 `Session Defaults`。

存储位置：

```text
~/.config/xenv/session/<session-id>.json
```

职责：

- 记录当前 shell 会话中用户通过 xenv 显式设置的 session 默认值。
- 作为后续 `xenv unuse`、`xenv unset`、`xenv path remove` 的参考。
- 为“离开目录后恢复之前 session 默认值”保留基线。
- 作为 xenv 多次子进程执行之间的会话记忆。

不负责：

- 不等于当前 shell 的 runtime 真相。
- 不应该因为 `init-direnv` 应用了 `.xenv.toml` 而被改写为目录 SDK。
- 不应该被 `xenv use -g` 或 `xenv use -s` 修改其 defaults。

当前结构可以暂时继续使用：

```json
{
  "sdks": {
    "go": "1.24.6"
  },
  "envs": {},
  "paths": [],
  "dir_states": {}
}
```

但语义上应解释为：

- `sdks` / `envs` / `paths`: 当前 shell 的 session defaults。
- `dir_states`: 当前 shell 关联或曾应用过的目录状态上下文，用于未来清理和恢复。

后续如果需要更清晰的结构，可以迁移为：

```json
{
  "defaults": {
    "sdks": {
      "go": "1.24.6"
    },
    "envs": {},
    "paths": []
  },
  "applied_direnv": {
    "file": "D:/project/.xenv.toml",
    "sdks": {
      "go": "1.23.9"
    },
    "envs": {},
    "paths": []
  }
}
```

这类结构迁移不是第一阶段目标。

### Effective State

Effective State 是当前目录下“xenv 期望生效”的状态。它不是持久化文件，而是根据多个状态源推导出来的结果。

优先级：

```text
Directory State > Session State > Global State
```

职责：

- 用于展示当前目录期望 SDK / ENV / PATH。
- 用于标注哪些低优先级状态被覆盖。
- 用于后续与 Runtime State 做差异检测。

不负责：

- 不直接修改 shell。
- 不直接保存到文件。
- 不保证一定等于当前 `go version` 的结果。

示例：

```text
Global State:
  go => 1.21.13

Session State:
  go => 1.24.6

Directory State:
  go => 1.23

Effective State:
  go => 1.23  (Directory State)
```

### Runtime State

Runtime State 是当前 shell 真实环境。它由当前 shell 中的 `PATH`、`GOROOT`、`JAVA_HOME`、其他 ENV 等决定。

职责：

- 表示用户现在执行 `go`、`java`、`node` 等命令时真实命中的状态。
- 由 shell hook eval `xenv` 输出的表达式后改变。
- 后续可以通过当前进程环境实时检测或估算。

不负责：

- 不应该只从 session JSON 得出。
- 不应该作为绝对事实持久化到 session JSON。
- 不应该被用来反向覆盖 `.xenv.toml` 或 global state。

Runtime State 的事实来源是 shell 环境，而不是 xenv 文件：

```text
PATH
GOROOT
JAVA_HOME
NODE_HOME
其他 ActiveEnv
```

如果未来需要持久化 runtime 相关信息，只能作为调试缓存，例如 `last_applied_runtime`，不能作为真相。

## 状态触发边界

### Session State 持久化边界

Session State 的持久化由 xenv 进程写文件触发。

触发条件：

```text
1. 当前在 xenv shell hook 环境中。
2. 操作目标是 session scope。
3. session state 被标记为 HasUpdate。
```

典型命令：

```bash
xenv use go:1.24
xenv unuse go
xenv set FOO bar
xenv unset FOO
xenv path add D:/tools/bin
xenv path remove D:/tools/bin
```

这些命令默认是 session scope，除非显式加 `-g` 或 `-s`。

### Runtime State 应用边界

Runtime State 的变化由 shell eval 执行脚本触发。

流程：

```text
1. 用户执行 xenv 命令。
2. shell hook 调用真实 xenv binary。
3. xenv 输出消息和 --Expression-- 分隔符。
4. shell hook 提取 --Expression-- 后面的脚本。
5. bash/zsh 执行 eval，pwsh 执行 Invoke-Expression。
6. 当前 shell 的 PATH/ENV 改变。
```

关键点：

```text
xenv 输出脚本 != runtime 已改变
shell hook eval 脚本 == runtime 改变
```

如果绕过 hook 执行真实 binary：

```bash
command xenv use go:1.24
```

则 xenv 可能会写状态文件，但当前 shell runtime 不一定变化。

### Directory State 应用边界

Directory State 的应用由 `init-direnv` 触发。

典型触发：

```bash
cd project
cd .
xenv init-direnv
```

行为：

```text
读取 .xenv.toml / .xenv.local.toml
生成 runtime expression
由 shell hook eval 后修改 Runtime State
不改写 session defaults
```

如果 `.xenv.toml` 已存在并声明了 SDK，`init-direnv` 应该应用它，但不应该把该 SDK 写入 `session.SDKs`。

## 命令语义

| 命令 | 持久化目标 | Runtime 是否立即变化 | 说明 |
| --- | --- | --- | --- |
| `xenv use go:1.24` | Session State | 是，hook eval 后变化 | 设置当前 shell 的 session default，同时立即激活。若当前目录有 Directory State，这是临时 runtime override。 |
| `xenv unuse go` | Session State | 是，hook eval 后变化 | 删除当前 shell 的 session default，并尝试从 runtime 移除对应 PATH/ENV。 |
| `xenv use -s go:1.24` | Directory State | 是，hook eval 后变化 | 修改项目约束，并立即激活。 |
| `xenv unuse -s go` | Directory State | 是，hook eval 后变化 | 修改项目约束，并立即从 runtime 移除对应配置。 |
| `xenv use -g go:1.24` | Global State | 是，hook eval 后变化 | 修改全局默认，并立即激活当前 shell。下次目录 hook 仍可被 Directory State 覆盖。 |
| `xenv init-direnv` | 不改 session defaults | 是，hook eval 后变化 | 应用当前目录的 Directory State 到 runtime。 |
| 用户手动修改 `PATH` | 无 | 是 | xenv 不知道这次变化，Runtime State 与 xenv state 可能脱节。 |

## 示例场景

### 场景一: 目录配置覆盖 session default

项目配置：

```toml
[sdks]
go = "1.23"
```

用户进入目录：

```bash
cd project
```

状态：

```text
Directory State:
  go => 1.23

Session State:
  空或其他值

Effective State:
  go => 1.23

Runtime State:
  go => 1.23.x
```

### 场景二: 在目录内执行普通 `xenv use`

用户执行：

```bash
xenv use go:1.24
```

状态：

```text
Directory State:
  go => 1.23

Session State:
  go => 1.24.x

Effective State:
  go => 1.23

Runtime State:
  go => 1.24.x
```

解释：

- `xenv use go:1.24` 默认修改 session default。
- 当前 shell runtime 会立即切换到 `go1.24.x`。
- `.xenv.toml` 没有被修改，所以 Effective State 仍然是 `go1.23`。
- 这属于临时 runtime override。

建议提示：

```text
Activate go:1.24.x for current session
WARN: directory state wants go:1.23; this activation is a temporary runtime override.
Use `xenv use -s go:1.24` to update .xenv.toml.
```

### 场景三: 重新触发 direnv

继续执行：

```bash
cd .
```

状态：

```text
Directory State:
  go => 1.23

Session State:
  go => 1.24.x

Effective State:
  go => 1.23

Runtime State:
  go => 1.23.x
```

解释：

- `cd .` 触发 `init-direnv`。
- Directory State 被重新应用到 Runtime State。
- Session State 仍保留 `go1.24.x`，作为 session default 和未来恢复基线。

## CLI 组织决策

`xenv` 仍处于 v0 开发阶段，不需要为了兼容保留语义不清晰的旧入口。状态诊断应从 `list/ls` 中拆出，成为一等命令：

```bash
xenv status
```

顶层 `list` / `ls` 不再保留。需要列表能力时使用更明确的子命令：

```bash
xenv sdk list
xenv env list
xenv path list
```

职责边界：

```text
use / unuse      修改 SDK 激活状态
env / path       修改 ENV/PATH 状态
sdk              管理和查询本机 SDK inventory
status           查看当前目录和当前 shell 的状态
check            校验 Effective State / 项目要求是否满足
shell            生成和安装 shell hook
config           管理配置
```

`check` 保留 pass/fail 语义，不承担状态解释职责。Runtime State 的解释性输出归 `status`。

## 展示建议

`xenv status` 后续应明确展示状态层级，不再让用户误以为 Session State 一定是当前 runtime。

默认 `xenv status` 输出当前最重要的信息：Effective State、Runtime State 和 warning。

建议输出：

```text
[Effective State]
---------------------------------------------------------------------
Active Develop SDKs:
          go => 1.23  (Directory State)

[Runtime State]
---------------------------------------------------------------------
Active Develop SDKs:
          go => 1.24.6  (detected from PATH)
Warnings:
          go runtime differs from Effective State go 1.23
          run `cd .` or `xenv init-direnv` to re-apply directory state

[Global State]
---------------------------------------------------------------------
 - from: C:\Users\inhere/.config/xenv/global.toml
No global state found

[Directory States]
---------------------------------------------------------------------
 - from: D:\work\project\.xenv.toml
Directory SDKs:
          go => 1.23

[Session Context]
---------------------------------------------------------------------
 - from: C:\Users\inhere/.config/xenv/session/xenv_xxx.json
Session Defaults:
          go => 1.24.6  (overridden by Directory State)
```

分层详情通过参数显式展开：

```bash
xenv status --layers
```

Runtime 检测详情通过参数显式展示：

```bash
xenv status --runtime
```

参数可以组合：

```bash
xenv status --layers --runtime
```

第一阶段应完成：

- 一等 `xenv status` 命令。
- Effective State 展示。
- `[Session Context]` 和 `Session Defaults` 展示。
- overridden 标注。
- 目录内普通 `xenv use` 与 Directory State 冲突时的临时 runtime override warning。
- 移除顶层 `list` / `ls` 入口。

Runtime State 检测可以作为第二阶段，但入口应预留在 `xenv status --runtime`，而不是 `xenv ls --group --runtime`。

## 设计原则

1. 状态文件表达 xenv 管理的配置和上下文，不等于 shell runtime 真相。
2. Runtime State 只由 shell 实际环境决定，检测时应从 PATH/ENV 推导。
3. Directory State 是项目约束，默认优先于普通 Session State。
4. Session State 是当前 shell 的 defaults/context/recovery baseline，不是 runtime fact。
5. 普通 `xenv use` 可以临时改变 Runtime State，但不修改 Directory State。
6. `xenv use -s` 才表示修改项目期望状态。
7. 展示层不能使用会修改 `HasUpdate` 的 merge 方法计算 Effective State。

## 对实现计划的影响

已有实施计划需要按本文档调整：

- 在计划中明确关联本设计文档。
- 将顶层状态诊断入口从 `xenv ls --group` 改为 `xenv status`。
- 将 `[Session State]` 展示标题调整为 `[Session Context]`，内部展示 `Session Defaults`。
- v0 阶段不保留顶层 `list` / `ls` 兼容入口。
- 第一阶段实现 Effective State、overridden 标注和 session override warning。
- Runtime State 检测保持第二阶段设计，不应从 session JSON 直接得出。
- 后续如果实现 Runtime Warning，应从当前 PATH/ENV 检测，而不是信任 session JSON。

## 已确认决策

1. 使用 `[Session Context]` 作为展示标题，内部字段使用 `Session Defaults`。
2. 为目录内普通 `xenv use` 与 Directory State 冲突的情况增加临时 runtime override warning。
3. 使用一等 `xenv status` 命令承载状态诊断；Runtime 检测详情使用 `xenv status --runtime`。
4. v0 阶段不保留顶层 `list` / `ls` 兼容入口。

## 仍需后续决策

1. 是否在 session JSON 中新增 `applied_direnv` 元数据，用于未来离开目录恢复和调试。
2. Runtime State 检测是否默认启用，还是只在 `xenv status --runtime` 时执行完整检测。
3. 是否为脚本和 IDE 集成增加 `xenv status --json`。
