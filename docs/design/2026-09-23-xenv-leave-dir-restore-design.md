<!-- template_id: design; template_version: 1.1.1 -->
# xenv 离开目录恢复会话默认值设计

> 状态：Draft 0.1 / 待人工计划批准

## 修订记录

| 版本 | 日期 | 作者 | 摘要 |
|---|---|---|---|
| 0.1 | 2026-09-23 | Pi | 初稿：定义 direnv 应用记录、离开目录的撤销语义、hook 触发方式与失败边界，并列出待确认事项。 |

> 仅语义变化递增版本；纯 identity/provenance/元数据纠正沿用原版本，并在 Git/进度记录中留痕。

## 规划可靠性声明

- `thinking_mode=RIGOROUS`。
- `core_objective`：在已启用 shell hook 的会话中，离开带 `.xenv.toml` 的目录时撤销该目录带来的 PATH/ENV/SDK 变更，恢复进入前的会话默认值。
- `allowed_scope`：`internal/xenv/models`（状态结构）、`internal/xenv/manager`（状态读写）、`internal/xenv/service`（应用/撤销编排）、`internal/cli`（命令与展示）、测试与双语文档。
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
| 应用记录 | 记录「本次 direnv 应用给 shell 带来的可撤销变更」的结构，保存在 session state 中 |
| 应用（apply） | 把 direnv 状态转换成 shell 脚本并 eval，使其在当前 shell 生效 |
| 撤销（leave） | 按应用记录反向操作，使 shell 回到应用之前的状态 |
| hook shell | 已加载 `xenv shell` 输出的 shell（`XENV_HOOK_SHELL` 非空） |

## 范围与非目标

范围：

1. 定义应用记录的结构、写入时机与生命周期（单条，只保留最近一次应用）。
2. 定义撤销算法（PATH 条目移除、ENV 值恢复/取消设置）与幂等规则。
3. 在 `xenv init-direnv` 内部实现「先撤销、后应用」，hook 模板与调用方式不变。
4. 让 direnv 作用域的命令（`use -s`、`set -s`、`path add/remove -s`）在 hook shell 中同步更新应用记录。
5. 失败边界、兼容旧 session 文件、`xenv status` 展示与文档。

非目标：

1. 不做整环境快照/回滚：只撤销应用记录中的条目，不还原用户在该目录内对其它变量的改动。
2. 不撤销 `source_project_scripts` 打开时 source 的项目脚本与 `.envrc` 的副作用（sourcing 不可逆，见“安全”）。
3. 不实现 `direnv allow` 类逐目录审批。
4. 不改 `.xenv.toml` 文件格式，不改 `-s` 系列命令的持久化语义，不改 `use`/`env`/`path` 的非 direnv 作用域行为。
5. 不支持多个 `.xenv.toml` 叠加（仍只加载最近一个），也不为非 hook shell 提供离开逻辑。

## 已确认事实与规范

| 事实 | 证据 |
|---|---|
| hook 在 cd 后调用 `xenv init-direnv`（四个 shell 模板一致） | `gen_hook_bash.go:51`、`gen_hook_zsh.go:44`、`gen_hook_pwsh.go:166`、`gen_hook_cmd.go:72` |
| 应用内容 = SDK bin 目录 + SDK active_env + direnv paths + direnv envs + 项目脚本 | `internal/xenv/service/sdk_service.go: SetupDirenv` |
| 脚本由 hook eval（`--Expression--` 之后的部分） | `internal/xenv/shell/util.go: OutputScript`、各模板的 `invoke_xenv_result` |
| 会话状态已含 `dir_states`，但只在 `UseSDKsWithParams` 写入，且无撤销用途 | `models/state.go:62,207-218`、`manager/state_manager.go:87`、`cli/status_cmd.go:62` |
| `AddEnvs`/`AddPath` 的 direnv 分支不写 `dir_states` | `manager/state_manager.go`（`OpFlagDirenv` 分支） |
| session state 只在 hook shell 中保存 | `manager/state_manager.go: SaveStateFile`（`xenvcom.InHookShell()` 门控） |
| 非 hook shell 下 `SetupDirenv` 直接返回空脚本并提示 | `internal/xenv/service/sdk_service.go: SetupDirenv` |
| 上一阶段已把 `.xenv.toml` 的写入位置、脚本引用与 `check_tools_on_direnv` 修好 | 本会话提交 `488de42`、`2630316`、`7c29284` |

规则引用：`SR1204`（范围确认）、`SR1207`（设计→计划→实施）、`SR1403`（最小实现）、`SR1407`（最小验证）、`SR1107`（大型调整走 DPI）、`GR106`（测试断言）。

偏离：无。

## 总体方案

### 应用记录

在 session state 中新增 `applied_direnv`（沿用既有状态语义设计的命名，替代当前语义不完整的 `dir_states` 用途）：

```json
{
  "applied_direnv": {
    "file": "D:/proj/.xenv.toml",
    "paths": ["D:/proj/bin"],
    "envs": [
      {"name": "APP_ENV", "prev": "dev", "had_prev": true},
      {"name": "GOROOT", "had_prev": false}
    ],
    "sdks": {"go": "1.24.13"},
    "applied_at": "2026-09-23T10:00:00+08:00"
  }
}
```

- `paths`：本次应用新加入 shell PATH 的条目（已在 PATH 中的条目不记录）。
- `envs`：本次应用设置过的变量名，`prev`/`had_prev` 记录应用前的值（`had_prev=false` 表示应用前未设置）。
- `sdks`：本次激活的 SDK，用于 `status` 展示与调试（撤销时按 bin 目录/active_env 处理，不单独使用）。
- 只保留最近一次应用；写入时机为「应用脚本生成后」。

### 撤销算法

1. 读取应用记录；不存在 → 无操作（兼容旧 session 文件）。
2. 记录中的 `file` 与即将应用的 `.xenv.toml` 相同 → 同一项目，跳过撤销与重复应用（幂等）。
3. 否则生成撤销脚本：从当前 shell PATH 中移除 `paths`（按平台规则归一化比较），对 `envs` 逐项恢复 `prev` 或 unset。
4. 撤销成功后清空记录，再按现有逻辑应用新的 direnv 状态并写入新记录。
5. 撤销脚本与应用脚本在同一次 `--Expression--` 输出中返回，hook 一次 eval（不新增 hook 命令、不改四个模板）。

### 记录写入

- `SetupDirenv`：应用前用当前进程环境（即 shell 环境）计算「新增的 PATH 条目」与「受影响变量及其旧值」，应用成功后写入记录。
- `use -s`、`set -s`、`path add/remove -s`：这些命令在 hook shell 中会直接把变更应用到当前 shell（direnv 作用域），因此复用同一记录逻辑追加 delta；写入点收敛到一个 helper，避免多处各写一份。

### 触发与命令面

- hook 仍只调用 `xenv init-direnv`（不改模板）：命令内部完成「撤销上一记录 + 应用当前目录」。
- 手动执行 `xenv init-direnv`（非 hook）保持现状：只应用当前目录，不做撤销（无 session 记录上下文）。
- 可选（待确认 Q2）：新增 `xenv leave-direnv` 供用户手动清理当前记录。

## 架构

```text
hook (cd 后)
   └─> xenv init-direnv
         └─> SDKService.SetupDirenv
               ├─ 读取 session 应用记录 (StateManager.AppliedDirenv)
               ├─ 生成撤销脚本 (GenSetPath / GenSetEnv / GenUnsetEnv)
               ├─ 应用当前 .xenv.toml (现有逻辑)
               └─ 写入新的应用记录 (StateManager.SetAppliedDirenv)
                     └─ session state 文件: applied_direnv
```

预计改动文件：

| 文件 | 类型 | 职责 |
|---|---|---|
| `internal/xenv/models/state.go` | 改动 | `AppliedDirenv` 结构、`AppliedEnv` 子结构、增删与判空方法 |
| `internal/xenv/manager/state_manager.go` | 改动 | 记录的读写与持久化；direnv 分支统一更新记录 |
| `internal/xenv/service/sdk_service.go` | 改动 | `SetupDirenv` 拆分「撤销 / 应用 / 记录」三步 |
| `internal/xenv/service/env_service.go` | 改动 | `-s` 系列命令追加 delta 到记录 |
| `internal/cli/status_cmd.go` | 改动 | `--layers` 展示当前应用记录（替换现有 `dir_states` 展示） |
| `internal/xenv/*/**_test.go` | 新增/改动 | 记录、撤销、幂等、兼容、失败边界 |
| `README.md`、`README.zh-CN.md` | 改动 | 说明离开目录行为与不可撤销项 |

## 关键流程

进入与离开：

```text
会话开始(无记录)
  cd projA   -> 无记录, 直接应用 projA; 记录 paths=[A/bin], envs=[APP_ENV: prev=dev]
  cd projA/x -> 记录 file 与 projA 相同, 跳过(幂等)
  cd projB   -> 撤销 projA(A/bin 移除, APP_ENV 恢复 dev); 应用 projB; 记录 projB
  cd ~       -> 撤销 projB(GOROOT 取消设置, ...); 记录清空
```

失败路径：

| 场景 | 行为 |
|---|---|
| session 文件不可写 | 只输出 WARN，不阻断 cd；本次不写入记录（下次进入会重复应用，可能重复 PATH，已在“安全”中说明） |
| `.xenv.toml` 解析失败 | 保持现状：WARN + 只输出撤销脚本（若存在记录） |
| 撤销时 PATH 中已无该条目 | 跳过该条目，不算错误 |
| 记录中的变量被用户在当前 shell 改过 | 按记录恢复为进入前的值（与 direnv 行为一致，见 Q1） |

## 安全、数据、运维与回滚

- 数据：只在 session state 中新增一个字段；不改 `.xenv.toml`、不改 global state；旧文件无需迁移（缺字段即视为无记录）。
- 安全：撤销只针对记录内的条目，不会批量清理用户自己的 PATH 条目；`source_project_scripts` 打开时的脚本副作用不可撤销（默认关闭，README 需明示）。
- 运维：无新增依赖、无网络、无后台进程；记录只在 hook shell 中写入，非 hook 环境行为不变。
- 回滚：删除 `applied_direnv` 字段并移除撤销步骤即可恢复现状；已写入的记录不会影响其它功能（`status` 之外无人读取）。

## 决策

| 编号 | 决策 | 理由与被否方案 |
|---|---|---|
| D1 | 撤销由 `xenv init-direnv` 内部完成，hook 模板与命令面不变 | 四个模板已经统一调用同一命令，内部完成即可；备选「新增 leave-direnv 命令并在四个模板中调用」要改 4 处模板且增加 hook 往返 |
| D2 | 只记录并撤销「本次应用带来的变更」（delta），不做整环境快照 | delta 足够覆盖 SDK/PATH/ENV 三类变更，且实现与状态体积都小；整环境快照需要 hook 保存完整 env，成本和风险更高（SR1403） |
| D3 | 记录保存在 session state 的 `applied_direnv`，单条，只保留最近一次 | 与既有状态语义设计一致；备选「独立文件」增加 IO 与清理负担 |
| D4 | 同一 `.xenv.toml` 之间切换（含子目录）跳过撤销与应用 | 避免重复追加 PATH 条目（幂等） |
| D5 | `-s` 系列命令在 hook shell 中同步追加记录 | 否则这些命令的变更会「应用了但记录不到」，离开目录时无法恢复 |
| D6 | 撤销失败只 WARN，不阻断 cd | cd 是高频交互动作，阻断代价高于残留风险；失败场景已在文档中说明 |
| D7 | `envs` 恢复策略：`had_prev=true` 恢复旧值，`had_prev=false` 取消设置 | 与 direnv 语义一致，避免把变量恢复成空字符串 |
| D8 | 用 `applied_direnv` 取代 `dir_states` 的撤销职责；`dir_states` 展示改为读取应用记录 | 当前 `dir_states` 写入不完整且无撤销用途，保留两套记录会产生歧义 |

## 待确认事项

| 编号 | 问题 | 备选与影响 |
|---|---|---|
| Q1 | 用户在该目录内手动修改了同名变量，离开时是否恢复进入前的值 | 默认恢复（与 direnv 一致）；若要保留用户改动，需要在记录中增加“最后写入值”比对，复杂度上升 |
| Q2 | 是否新增 `xenv leave-direnv` 手动命令 | 便于用户主动清理；不实现时可用 `cd` 到无配置目录触发撤销 |
| Q3 | `xenv status` 是否展示应用记录（`--layers` 与默认输出） | 影响展示契约；不展示时用户只能通过 PATH/ENV 观察 |
| Q4 | 项目脚本/`.envrc` 副作用不可撤销是否可接受 | 默认接受（`source_project_scripts` 默认关闭）；若要可撤销需改为“子 shell 隔离”方案，属于另一设计 |
| Q5 | 记录字段命名与位置（session state vs 独立文件） | 默认 session state（D3）；独立文件更易清理但增加 IO |

## 结论与人工计划 Gate

本设计把「离开目录」定义为按应用记录撤销上一次 direnv 应用，并给出记录结构、撤销算法、幂等规则与失败边界；hook 命令面与四个模板不变，兼容旧 session 文件，非 hook 环境行为不变。

Delivery Track 判定为 Full（改变状态语义与跨模块流程），治理暴露度低（本地开发工具、无生产数据、无出机器动作），评审封顶一轮合并轴评审。实施前需要设计批准，并按 `SR1204` 完成范围确认（预计 6-7 个文件、约 250-350 行业务代码）。

待批准语句（可直接复制）：

```text
批准 docs/design/2026-09-23-xenv-leave-dir-restore-design.md 的设计；
待确认事项 Q1-Q5 中未确认的按设计默认值处理。
```

批准只授权进入计划阶段；实施需要实施计划的人工批准，评审结论不构成批准。
