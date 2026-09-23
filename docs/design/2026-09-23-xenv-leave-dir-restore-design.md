<!-- template_id: design; template_version: 1.1.1 -->
# xenv 离开目录恢复会话默认值设计

> 状态：Draft 0.2 / 待人工计划批准

## 修订记录

| 版本 | 日期 | 作者 | 摘要 |
|---|---|---|---|
| 0.1 | 2026-09-23 | Pi | 初稿：定义 direnv 应用记录、离开目录的撤销语义、hook 触发方式与失败边界，并列出待确认事项。 |
| 0.2 | 2026-09-23 | Pi | 应用记录载体由 session state 改为 shell 环境变量 `XENV_APPLIED_DIRENV`（用户选定方案 A）：D3 重写、Q5 关闭，补充记录的格式、大小与生命周期约束，并记录 session 文件按目录键控的实测证据。 |

> 仅语义变化递增版本；纯 identity/provenance/元数据纠正沿用原版本，并在 Git/进度记录中留痕。

## 规划可靠性声明

- `thinking_mode=RIGOROUS`。
- `core_objective`：在已启用 shell hook 的会话中，离开带 `.xenv.toml` 的目录时撤销该目录带来的 PATH/ENV/SDK 变更，恢复进入前的会话默认值。
- `allowed_scope`：`internal/xenv/models`（记录结构）、`internal/xenv/xenvcom`（记录变量名常量）、`internal/xenv/service`（应用/撤销/记录编排）、`internal/cli`（命令与展示）、测试与双语文档。
- `non_goals`：见“范围与非目标”。
- `expansion_policy=DEFER_OR_REQUEST`：scope freeze 后新增的 Module、外部协议、生命周期动作只在直接追溯核心验收、安全或可执行唯一性时进入候选，否则记为 deferred 或请求扩围。
- review budget：Full track，低暴露度（本地开发工具、无生产数据、无出机器动作），评审封顶一轮合并轴评审，不建 review ledger。
- 停止条件：出现改变用户结果、接口、安全或授权的核心选择时停止并请求确认；无 `CORE_BLOCKING` 时在本设计的人工计划 Gate 停止。

## 背景与目标

现状（已确认）：

- shell hook 在每次 `cd` 后调用 `xenv init-direnv`（`internal/xenv/shell/gen_hook_bash.go:51`、`gen_hook_zsh.go:44`、`gen_hook_pwsh.go:166`、`gen_hook_cmd.go:72`），由 `SDKService.SetupDirenv`（`internal/xenv/service/sdk_service.go`）把最近一个 `.xenv.toml` 的 SDK/PATH/ENV 追加到当前 shell。
- 只有「进入」逻辑，没有「离开」逻辑：`pwsh` 模板里离开分支仍是注释 TODO（`gen_hook_pwsh.go:146-147`），bash/zsh 的 `cd` 包装只调用 `init-direnv`。
- 会话状态里已有 `dir_states` 字段（`internal/xenv/models/state.go:62`），但只在 `UseSDKsWithParams` 的 direnv 分支写入（`internal/xenv/manager/state_manager.go:87`），`AddEnvs`/`AddPath` 的 direnv 分支不写；该字段目前只被 `xenv status --layers` 读取（`internal/cli/status_cmd.go:62`），没有任何撤销/回滚用途。
- 既有设计已明确把它排除在第一阶段之外：`docs/design/2026-06-12-xenv-state-semantics-design.md`（“本文档不解决‘离开目录自动恢复 session default’的完整生命周期实现”）。

0.1 起草后、实施 T3 时发现的阻塞事实（决定 0.2 的修订）：

- 四个 hook 模板都会**清除**会话 ID：`unset XENV_SESSION_ID`（`gen_hook_bash.go:98`、`gen_hook_zsh.go:87`）、`Remove-Item Env:XENV_SESSION_ID`（`gen_hook_pwsh.go:178`）、`os.setenv("XENV_SESSION_ID", nil)`（`gen_hook_cmd.go:85`）；`xenvcom.SessionID()`（`internal/xenv/xenvcom/var.go:19`）在变量为空时回退为「按当前目录的项目根」生成。
- 实测（同一 HOME、同一 hook 环境，先 `init-direnv` 进目录 A 再进目录 B）：生成两个不同的 session 文件 `sessA_a4c9093180.json`、`sessB_d99ff0e600.json`。
- 由于 hook 在 `cd` **之后**调用 `xenv init-direnv`，进程读的是新目录的 session 文件，读不到上一个目录写入的记录 → 0.1 的 D3（记录放 session state）无法满足验收 1/3。

用户可见问题：

1. 进入项目目录后，该项目的 PATH 条目与 ENV 会一直留在 shell 里；离开目录后仍然生效，可能遮蔽同名命令或污染后续目录。
2. 连续进入多个项目时，上一个项目的 PATH/ENV 会累积叠加，无法回到「未进入任何项目」的状态。
3. 用户只能新开终端（或手工 `xenv path remove` / `xenv env unset`）来清理。

目标（用户可见结果）：

```bash
cd ~/projA        # 应用 projA 的 .xenv.toml
go version        # 使用 projA 指定的 go
cd ~/projB        # 先撤销 projA 的变更, 再应用 projB
cd ~              # 撤销 projB 的变更, 回到进入前的会话默认值
```

## 名词

| 名词 | 含义 |
|---|---|
| direnv 状态 | 最近一个 `.xenv.toml` 解析出的 `ActivityState`（`Paths`/`SDKs`/`Envs`/`ToolRequirements`） |
| 会话默认值 | 进入任何项目目录之前，当前 shell 自身的 PATH/ENV（不含任何 direnv 变更） |
| 应用记录 | 记录「本次 direnv 应用给 shell 带来的可撤销变更」的结构 |
| 记录变量 | 承载应用记录的 shell 环境变量 `XENV_APPLIED_DIRENV`（JSON 文本） |
| 应用（apply） | 把 direnv 状态转换成 shell 脚本并 eval，使其在当前 shell 生效 |
| 撤销（leave） | 按应用记录反向操作，使 shell 回到应用之前的状态 |
| hook shell | 已加载 `xenv shell` 输出的 shell（`XENV_HOOK_SHELL` 非空） |

## 范围与非目标

范围：

1. 定义应用记录的结构，以及它由 shell 环境变量承载的写入/读取/清除时机。
2. 定义撤销算法（PATH 条目移除、ENV 值恢复/取消设置）与幂等规则。
3. 在 `xenv init-direnv` 内部实现「先撤销、后应用」，hook 模板与调用方式不变。
4. 让 direnv 作用域的命令（`use -s`、`set -s`、`path add/remove -s`）在 hook shell 中同步更新记录。
5. 失败边界、兼容旧环境（无记录变量）、`xenv status` 展示与文档。

非目标：

1. 不做整环境快照/回滚：只撤销应用记录中的条目，不还原用户在该目录内对其它变量的改动。
2. 不撤销 `source_project_scripts` 打开时 source 的项目脚本与 `.envrc` 的副作用（sourcing 不可逆，见“安全”）。
3. 不实现 `direnv allow` 类逐目录审批。
4. 不改 `.xenv.toml` 文件格式，不改 `-s` 系列命令的持久化语义，不改 `use`/`env`/`path` 的非 direnv 作用域行为。
5. 不改四个 hook 模板（记录随应用脚本注入 shell，而不是靠模板传递）；不改 session 文件按目录键控的现状。
6. 不支持多个 `.xenv.toml` 叠加（仍只加载最近一个），也不为非 hook shell 提供离开逻辑。

## 已确认事实与规范

| 事实 | 证据 |
|---|---|
| hook 在 cd 后调用 `xenv init-direnv`（四个 shell 模板一致） | `gen_hook_bash.go:51`、`gen_hook_zsh.go:44`、`gen_hook_pwsh.go:166`、`gen_hook_cmd.go:72` |
| 应用内容 = SDK bin 目录 + SDK active_env + direnv paths + direnv envs + 项目脚本 | `internal/xenv/service/sdk_service.go: SetupDirenv` |
| 脚本由 hook eval（`--Expression--` 之后的部分），因此脚本内 `export` 的变量会留在该 shell | `internal/xenv/shell/util.go: OutputScript`、各模板的 `invoke_xenv_result` |
| hook 会清除 `XENV_SESSION_ID`，session 文件按当前目录的项目根键控 | `gen_hook_bash.go:98` 等四处；`xenvcom/var.go:19`；实测 A/B 两个 session 文件 |
| session state 只在 hook shell 中保存 | `manager/state_manager.go: SaveStateFile`（`xenvcom.InHookShell()` 门控） |
| 非 hook shell 下 `SetupDirenv` 直接返回空脚本并提示 | `internal/xenv/service/sdk_service.go: SetupDirenv` |
| 脚本片段的引号/转义已统一（含单引号、空格、Git Bash 路径） | 本会话提交 `c3a6e75`、`2032139` |

规则引用：`SR1204`（范围确认）、`SR1207`（设计→计划→实施）、`SR1403`（最小实现）、`SR1407`（最小验证）、`SR1107`（大型调整走 DPI）、`GR106`（测试断言）。

偏离：无。

## 总体方案

### 应用记录

记录结构（存放在 `internal/xenv/models`，载体无关）：

```json
{
  "file": "D:/proj/.xenv.toml",
  "paths": ["D:/proj/bin"],
  "envs": [
    {"name": "APP_ENV", "prev": "dev", "had_prev": true},
    {"name": "GOROOT", "had_prev": false}
  ],
  "sdks": {"go": "1.24.13"},
  "applied_at": "2026-09-23T10:00:00+08:00"
}
```

- `paths`：本次应用新加入 shell PATH 的条目（已在 PATH 中的条目不记录）。
- `envs`：本次应用设置过的变量名，`prev`/`had_prev` 记录应用前的值（`had_prev=false` 表示应用前未设置）。
- `sdks`：本次激活的 SDK，用于展示与调试。
- 只保留最近一次应用；`IsEmpty()` 为真（无任何条目）时不写记录。

### 记录载体：shell 环境变量

- 变量名 `XENV_APPLIED_DIRENV`，值是上述 JSON 文本；常量放在 `internal/xenv/xenvcom`（与 `HookShellEnvName` 并列）。
- 写入：应用脚本末尾追加 `export XENV_APPLIED_DIRENV='<json>'`（由既有 `GenSetEnv` 生成，转义规则已统一）；脚本由 hook eval 后变量留在该 shell 内，因此**天然 per-shell**，且不依赖 session 文件、不需要修改任何 hook 模板。
- 读取：`xenv` 进程从继承的环境读取该变量并解析 JSON；空值或解析失败视为「无记录」（解析失败时输出一次 WARN）。
- 清除：撤销脚本末尾追加 `unset XENV_APPLIED_DIRENV`（`GenUnsetEnv`）。
- 约束：记录只包含路径与变量名/旧值，实测规模在数百字节量级；不写入任何敏感信息（不含变量当前值以外的内容）。

### 撤销算法

1. 读取记录变量；无记录（或解析失败）→ 无操作。
2. 记录中的 `file` 与即将应用的 `.xenv.toml` 相同 → 同一项目，跳过撤销与重复应用（幂等）。
3. 否则生成撤销脚本：从当前 shell PATH 中移除 `paths`（按平台规则归一化比较），对 `envs` 逐项恢复 `prev` 或 unset；脚本末尾清除记录变量。
4. 再按现有逻辑应用新的 direnv 状态，并在脚本末尾写入新的记录变量。
5. 撤销脚本与应用脚本在同一次 `--Expression--` 输出中返回，hook 一次 eval（不新增 hook 命令、不改四个模板）。

### 记录写入

- `SetupDirenv`：应用前用当前进程环境（即 shell 环境）计算「新增的 PATH 条目」与「受影响变量及其旧值」，应用成功后把记录写进脚本。
- `use -s`、`set -s`、`path add/remove -s`：这些命令在 hook shell 中会直接把变更应用到当前 shell（direnv 作用域），因此读取当前记录、合并本次 delta（同名变量保留最早的旧值）后重新写入脚本；写入点收敛到一个 helper。

### 触发与命令面

- hook 仍只调用 `xenv init-direnv`（不改模板）：命令内部完成「撤销上一记录 + 应用当前目录」。
- 手动执行 `xenv init-direnv`（非 hook）保持现状：只应用当前目录，不做撤销（无记录变量上下文）。
- 可选（待确认 Q2）：新增 `xenv leave-direnv` 供用户手动清理当前记录。

## 架构

```text
hook (cd 后)
   └─> xenv init-direnv
         └─> SDKService.SetupDirenv
               ├─ 读取记录变量 XENV_APPLIED_DIRENV (os.Getenv + JSON)
               ├─ 生成撤销脚本 (GenSetPath / GenSetEnv / GenUnsetEnv)
               ├─ 应用当前 .xenv.toml (现有逻辑)
               └─ 在脚本末尾写入新的记录变量 (GenSetEnv)
                     └─ hook eval 后留在该 shell, 供下一次 cd 读取
```

预计改动文件：

| 文件 | 类型 | 职责 |
|---|---|---|
| `internal/xenv/models/state.go` | 改动（0.1 已交付） | `AppliedDirenv`/`AppliedEnv` 结构与增删方法 |
| `internal/xenv/xenvcom/const.go` | 改动 | 新增记录变量名常量 `AppliedDirenvEnvName` |
| `internal/xenv/service/sdk_service.go` | 改动 | `SetupDirenv` 拆分「撤销 / 应用 / 记录」；记录读写与环境变量互转 |
| `internal/xenv/service/env_service.go` | 改动 | `-s` 系列命令合并 delta 并写入记录变量 |
| `internal/cli/status_cmd.go` | 改动 | `--layers` 展示当前应用记录（改为读取记录变量） |
| `internal/xenv/models/state.go`、`internal/xenv/manager/state_manager.go` | 改动（回退） | 移除 0.1 引入的 session state 承载（`ActivityState.AppliedDirenv` 字段与 `StateManager` 访问器）及其测试 |
| `internal/xenv/*/**_test.go` | 新增/改动 | 记录编解码、撤销、幂等、兼容（无记录变量）、解析失败 |
| `README.md`、`README.zh-CN.md` | 改动 | 说明离开目录行为与不可撤销项 |

## 关键流程

进入与离开：

```text
会话开始(无记录变量)
  cd projA   -> 无记录, 直接应用 projA; 脚本末尾写入记录(paths=[A/bin], envs=[APP_ENV: prev=dev])
  cd projA/x -> 记录 file 与 projA 相同, 跳过(幂等)
  cd projB   -> 撤销 projA(A/bin 移除, APP_ENV 恢复 dev, unset 记录变量); 应用 projB; 写入 projB 记录
  cd ~       -> 撤销 projB(GOROOT 取消设置, ...); unset 记录变量
```

失败路径：

| 场景 | 行为 |
|---|---|
| 记录变量解析失败 | 输出 WARN 并视为无记录（本次不撤销，进入新目录正常应用） |
| 记录变量超出环境变量长度限制 | 记录只保存必要条目；若写入失败由 shell 报错，xenv 侧不阻断（已在“安全”说明） |
| `.xenv.toml` 解析失败 | 保持现状：WARN + 只输出撤销脚本（若存在记录） |
| 撤销时 PATH 中已无该条目 | 跳过该条目，不算错误 |
| 记录中的变量被用户在当前 shell 改过 | 按记录恢复为进入前的值（与 direnv 行为一致，见 Q1） |

## 安全、数据、运维与回滚

- 数据：不改 `.xenv.toml`、不改 global state、不改 session state（0.1 引入的 session 承载将被回退）；记录只存在于当前 shell 环境。
- 安全：撤销只针对记录内的条目，不会批量清理用户自己的 PATH 条目；记录变量会被子进程继承（仅含路径与变量名/旧值，不含敏感值）；`source_project_scripts` 打开时的脚本副作用不可撤销（默认关闭，README 需明示）。
- 运维：无新增依赖、无网络、无后台进程；记录随 shell 生命周期存在，shell 退出即消失，无需清理。
- 回滚：移除记录变量的读写与撤销步骤即可恢复现状；记录变量对其它功能无影响（仅 `status` 读取）。

## 决策

| 编号 | 决策 | 理由与被否方案 |
|---|---|---|
| D1 | 撤销由 `xenv init-direnv` 内部完成，hook 模板与命令面不变 | 四个模板已经统一调用同一命令，内部完成即可；备选「新增 leave-direnv 命令并在四个模板中调用」要改 4 处模板且增加 hook 往返 |
| D2 | 只记录并撤销「本次应用带来的变更」（delta），不做整环境快照 | delta 足够覆盖 SDK/PATH/ENV 三类变更，且实现与状态体积都小；整环境快照需要 hook 保存完整 env，成本和风险更高（SR1403） |
| D3 | 记录由 shell 环境变量 `XENV_APPLIED_DIRENV` 承载，随应用脚本写入、由撤销脚本清除 | session 文件按当前目录键控（见“背景”的实测），而 hook 在 cd 之后调用，读不到上一个目录的记录，故 0.1 的「记录放 session state」不可实现。环境变量天然 per-shell 且**不需要改模板**；被否方案：稳定 `XENV_SESSION_ID`（改 4 个模板、session 文件增殖）、hook 传上一个目录（改模板且依赖 prev-dir 追踪） |
| D4 | 同一 `.xenv.toml` 之间切换（含子目录）跳过撤销与应用 | 避免重复追加 PATH 条目（幂等） |
| D5 | `-s` 系列命令在 hook shell 中同步合并记录 | 否则这些命令的变更会「应用了但记录不到」，离开目录时无法恢复 |
| D6 | 撤销失败只 WARN，不阻断 cd | cd 是高频交互动作，阻断代价高于残留风险；失败场景已在文档中说明 |
| D7 | `envs` 恢复策略：`had_prev=true` 恢复旧值，`had_prev=false` 取消设置 | 与 direnv 语义一致，避免把变量恢复成空字符串 |
| D8 | 用记录变量取代 `dir_states` 的撤销职责；`status` 展示改为读取记录变量 | 当前 `dir_states` 写入不完整且无撤销用途，保留两套记录会产生歧义 |
| D9 | 记录为 JSON 文本，变量名与常量集中在 `xenvcom` | 便于 `status`/调试查看，且与既有 `XENV_*` 变量命名一致；备选「自定义紧凑格式」无收益 |

## 待确认事项

| 编号 | 问题 | 备选与影响 |
|---|---|---|
| Q1 | 用户在该目录内手动修改了同名变量，离开时是否恢复进入前的值 | 默认恢复（与 direnv 一致）；若要保留用户改动，需要在记录中增加“最后写入值”比对，复杂度上升 |
| Q2 | 是否新增 `xenv leave-direnv` 手动命令 | 便于用户主动清理；不实现时可用 `cd` 到无配置目录触发撤销 |
| Q3 | `xenv status` 是否展示应用记录（`--layers` 与默认输出） | 影响展示契约；不展示时用户只能通过 PATH/ENV 观察 |
| Q4 | 项目脚本/`.envrc` 副作用不可撤销是否可接受 | 默认接受（`source_project_scripts` 默认关闭）；若要可撤销需改为“子 shell 隔离”方案，属于另一设计 |

Q5（记录存放位置）已在 0.2 关闭：由 D3 决定为 shell 环境变量，编号保留以便追溯。

## 结论与人工计划 Gate

本设计把「离开目录」定义为按应用记录撤销上一次 direnv 应用，并给出记录结构、撤销算法、幂等规则与失败边界；记录由 shell 环境变量承载（D3），hook 命令面与四个模板不变，非 hook 环境行为不变。

批准证据：用户 2026-09-23 批准设计 0.1 并确认 Q1-Q5 默认值；同日实施 T3 时发现 D3 不可实现，用户在三个候选中选定方案 A（记录改为 shell 环境变量），本 0.2 修订据此更新。批准只授权进入计划；实施需要实施计划的人工批准。

Delivery Track 判定为 Full（改变状态语义与跨模块流程），治理暴露度低（本地开发工具、无生产数据、无出机器动作），评审封顶一轮合并轴评审。实施前需按 `SR1204` 完成范围确认（预计 6-7 个文件、约 250-350 行业务代码）。

待批准语句（可直接复制）：

```text
批准 docs/design/2026-09-23-xenv-leave-dir-restore-design.md 0.2 的设计；
待确认事项 Q1-Q4 中未确认的按设计默认值处理。
```

批准只授权进入计划阶段；实施需要实施计划的人工批准，评审结论不构成批准。
