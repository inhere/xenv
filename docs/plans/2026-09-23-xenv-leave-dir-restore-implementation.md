<!-- template_id: plan; template_version: 1.2.0 -->
# xenv 离开目录恢复会话默认值实施计划

> 状态：Draft 0.2 / 待人工计划批准

## 修订记录

| 版本 | 日期 | 作者 | 摘要 |
|---|---|---|---|
| 0.1 | 2026-09-23 | Pi | 初稿，按已批准设计 0.1 拆分波次与任务，记录 workspace baseline、验证方式与人工 Gate。 |
| 0.2 | 2026-09-23 | Pi | 同步设计 0.2：记录载体由 session state 改为 shell 环境变量 `XENV_APPLIED_DIRENV`；T2 改为「记录编解码与承载」，新增回退 0.1 session 承载的任务，T3/T5 的存储读取同步调整。 |

> 仅语义变化递增版本；纯 identity/provenance/元数据纠正沿用原版本，并在 Git/进度记录中留痕。

## 目标与完成定义

目标：按已批准设计 `docs/design/2026-09-23-xenv-leave-dir-restore-design.md`（0.2）实现「离开目录撤销 direnv 变更」：hook 仍在 cd 后调用 `xenv init-direnv`，由命令内部先撤销上一次应用、再应用当前目录；应用记录由 shell 环境变量 `XENV_APPLIED_DIRENV` 承载（D3），不写 session state、不改 hook 模板。

完成定义（可观察）：

1. hook shell 中进入含 `.xenv.toml`（paths/envs/sdks）的目录后，PATH/ENV/SDK 生效；`cd` 到无配置目录后，该目录新增的 PATH 条目消失、受影响变量恢复进入前的值（原先未设置的变为未设置）。
2. 进入项目目录后再进入其子目录（同一 `.xenv.toml`）不重复应用，PATH 不出现重复条目。
3. 项目 A → 项目 B：A 的变更被撤销、B 生效。
4. 环境中没有记录变量（首次进入、旧 shell）时不报错，正常应用并写入记录。
5. 记录变量内容损坏（非法 JSON）时只 WARN，视为无记录，不阻断 cd。
6. 非 hook shell 执行 `xenv init-direnv` 行为与当前一致（只应用、不撤销）。
7. `xenv status --layers` 展示当前应用记录（读取记录变量）。
8. `go test ./...` 全绿；新增单测覆盖记录编解码、撤销、幂等、无记录与损坏记录。

## 范围、排除项与授权

范围：设计 0.2 的范围条目 1-5 与决策 D1-D9。

排除项：设计中的非目标 1-6；待确认事项 Q1-Q4 按默认值处理（Q1 恢复进入前的值、Q2 不新增 `leave-direnv`、Q3 仅在 `--layers` 展示、Q4 接受脚本副作用不可撤销）。

授权边界：仅本地代码改动、本地测试与本地 atomic commit（`SR1211`）；不 push/tag/release/部署；不改 `.xenv.toml` 与 global state 的既有语义；不修改四个 hook 模板。

- `host_or_non_offline_action=NOT_APPLICABLE`

## 输入与批准证据

- 设计：`docs/design/2026-09-23-xenv-leave-dir-restore-design.md`（0.2）。
- 设计 candidate：`git_root=D:\work\inhere\my-tools-dev\inhere-tools\xenv`、`commit=cf98c133911cb0837495083f99c7bc5d50aedeb3`、`subject_path=docs/design/2026-09-23-xenv-leave-dir-restore-design.md`、`document_revision=0.2`。
- 批准证据：用户 2026-09-23 批准设计 0.1 并确认 Q1-Q5 默认值；同日在 D3 不可实现的三个候选中选定方案 A（记录改 shell 环境变量），并说明“更新设计 0.2/计划再继续”。
- 规范：`SR1204`、`SR1207`、`SR1403`、`SR1407`、`SR1107`、`SR1206`、`SR1211`、`GR104`、`GR106`。

## 工作区基线

| 项 | 值 |
|---|---|
| Git root | `D:\work\inhere\my-tools-dev\inhere-tools\xenv` |
| Branch | `main` |
| HEAD | `cf98c133911cb0837495083f99c7bc5d50aedeb3` |
| `git status --short` | 空（无 in-scope 或 unrelated dirty/untracked） |
| 保留的无关 dirty 路径 | 无 |
| 依赖与能力 | Go 1.25.10（`go.mod` 要求 1.24）；无新增第三方依赖；端到端验证需要已安装 SDK（`xenv sdk list` 可见 go）与可用的 hook shell |
| 授权与生命周期边界 | 本地实现 + 本地验证 + 本地 atomic commit；无出机器动作 |
| Expected paths | 见各任务“文件”；Expected symbols：`models.AppliedDirenv`、`models.AppliedEnv`、`xenvcom.AppliedDirenvEnvName`、`SDKService.SetupDirenv`、`SDKService.loadAppliedRecord`、`SDKService.writeAppliedRecord`、`EnvService.recordAppliedDelta` |

## Capability Discovery

### Capability decisions

| capability_id | required_capability | searched_candidates | direct_reuse | thin_adapter_or_owner_extension | decision | proven_gap | duplication_and_lifecycle_risk |
|---|---|---|---|---|---|---|---|
| CAP-01 | 应用记录的结构与承载（环境变量 + JSON 编解码） | `models/state.go: AppliedDirenv`（0.1 已交付结构）；`xenvcom/const.go` 的 `XENV_*` 变量常量；`os.Getenv`/`encoding/json`；`shell.XenvScriptGenerator.GenSetEnv/GenUnsetEnv` | 复用 0.1 的记录结构、既有 `XENV_*` 常量风格与脚本生成器 | 在 `xenvcom` 增加变量名常量；在 service 增加 load/write 两个函数完成 JSON 互转 | OWNER_EXTENSION | none | 不得同时保留 session state 承载（0.1 的字段与访问器必须回退），否则出现两套记录来源 |
| CAP-02 | 撤销脚本生成（PATH 移除、变量恢复/取消设置） | `shell.XenvScriptGenerator: GenSetPath/GenSetEnv/GenUnsetEnv`；`util.SplitPath/JoinPaths`；`service: sessionPath/withoutPath` | 直接复用既有生成器与 PATH 工具函数 | 无 | DIRECT_REUSE | none | 不得另写一套引号/转义或 PATH 比较逻辑（本会话刚统一过） |
| CAP-03 | 应用/撤销编排与记录写入 | `service/sdk_service.go: SetupDirenv`（0.1 checkpoint 已拆分）；`service/env_service.go: SetEnvs/UnsetEnvs/AddPath/RemovePath` | 复用 0.1 的拆分结构与 `activateSDKs` 返回的激活参数 | 记录读写改为环境变量；`-s` 命令合并 delta 收敛到单一 helper | OWNER_EXTENSION | none | 记录写入若分散在多个命令里会漏记，收敛到 helper |
| CAP-04 | 展示当前应用记录 | `cli/status_cmd.go: buildEffectiveSDKRows/formatLayerLines` | 既有展示函数与分层输出 | `--layers` 改为读取记录变量 | THIN_ADAPTER | 现有展示读取 `dir_states`，语义与撤销记录不同 | 展示与撤销必须读同一份数据，避免用户看到不一致 |
| CAP-05 | 测试支撑（hook shell、环境变量注入） | `service` 包的 `newDirenvTestService`；`manager`/`cli` 包既有测试模式 | 直接复用既有 helper | 用 `t.Setenv` 注入/清理记录变量 | DIRECT_REUSE | none | 新测试不得依赖 ambient PATH/HOME/记录变量（本会话已修过同类问题） |

### New module candidates

| candidate_id | capability_id | proposed_module | searched_candidates | direct_reuse_gap | thin_adapter_or_owner_extension_gap | proven_gap | unique_owner_and_lifecycle | deletion_or_merge_handling |
|---|---|---|---|---|---|---|---|---|
| NONE | none | none | none | none | none | none | none | none |

### Rejected new tools

| rejected_candidate | capability_id | deletion_test_and_reason |
|---|---|---|
| 整环境快照回滚（保存完整 PATH/ENV 并在离开时整体还原） | CAP-03 | 删除测试：delta 记录已覆盖设计验收 1-3 的全部可撤销项；快照方案需要 hook 额外保存完整环境，状态体积与失败面更大 |
| 新增 `leave-direnv` 命令并在四个 hook 模板中调用 | CAP-03 | 删除测试：由 `init-direnv` 内部撤销即可满足验收，且不必改 4 个模板、不增加 hook 往返 |
| session state 承载记录（0.1 的 D3） | CAP-01 | 删除测试：session 文件按当前目录键控，cd 之后读取不到上一个目录的记录（实测），环境变量承载已满足验收且无需 session 写入；保留会造成两套记录来源 |

## 前置检查与 fail-closed 条件

前置检查：

1. `git status --short` 为空且 HEAD 为 `cf98c13`；否则先确认 dirty 归属。
2. `go build ./...`、`go vet ./...`、`staticcheck ./...` 通过。
3. 设计 0.2 已确认（方案 A）且 candidate 与上表一致。
4. 端到端验证需要：已安装的 go SDK（`xenv sdk list` 可见）、可在 Windows 上构造临时 HOME 与 hook 环境。

fail-closed 条件（命中即停止并回到 design/plan Gate）：

- 需要修改四个 hook 模板或新增 hook 命令才能满足验收（与 D1 冲突）。
- 需要改变 `.xenv.toml`、global state 或 `-s` 的持久化语义。
- 需要引入新依赖或新包。
- 记录变量超出 shell/环境变量长度限制而无法承载（需要回到 D3 重新选择）。
- 命中他人在途修改（Ownership Conflict）。

## 波次与依赖

| 波次 | 内容 | 任务 | 依赖 |
|---|---|---|---|
| W1 | 记录结构与承载 | T1（已完成）、T2、T2b | 设计 0.2 |
| W2 | 撤销与应用编排 | T3、T4 | W1 |
| W3 | 展示、文档与端到端验证 | T5、T6、T7 | W2 |

## 任务

### T1 应用记录结构（已完成）

- 文件: `internal/xenv/models/state.go`（已完成于 `3735ca2`）
- 动作: `AppliedDirenv`/`AppliedEnv` 结构与 `AddAppliedPath/AddAppliedEnv/AddAppliedSDK/RemoveAppliedEnv/IsEmpty`。
- 验证: `go test ./internal/xenv/models/ -count=1`。
- 完成标准: 结构可 JSON 往返，且不参与 `ActivityState.IsEmpty()`。
- 依赖: 无。

### T2 记录的环境变量承载

- 文件: `internal/xenv/xenvcom/const.go`（改动）、`internal/xenv/service/sdk_service.go`（改动）、`internal/xenv/service/tool_service_test.go`（改动）
- 动作:
  1. `xenvcom` 增加 `AppliedDirenvEnvName = "XENV_APPLIED_DIRENV"`。
  2. service 增加 `loadAppliedRecord() *models.AppliedDirenv`（读取 `os.Getenv`，空值返回 nil；JSON 解析失败输出一次 WARN 并返回 nil）与 `writeAppliedRecord(gen, rec)`（`IsEmpty` 时返回 `GenUnsetEnv` 行，否则返回 `GenSetEnv(name, json)` 行）。
  3. 单测覆盖：无变量、合法 JSON、非法 JSON、写入/清除的脚本行内容。
- 验证: `go test ./internal/xenv/service/ -count=1`；`go build ./... && go vet ./...`。
- 完成标准: 记录可经环境变量完整往返，损坏内容不阻断流程。
- 依赖: T1。

### T2b 回退 0.1 的 session state 承载

- 文件: `internal/xenv/models/state.go`（改动）、`internal/xenv/manager/state_manager.go`（改动）、`internal/xenv/manager/state_manager_test.go`（改动）、`internal/xenv/models/state_test.go`（改动）
- 动作:
  1. 移除 `ActivityState.AppliedDirenv` 字段与 `SetAppliedDirenv/ClearAppliedDirenv/HasAppliedDirenv` 方法。
  2. 移除 `StateManager.AppliedDirenv/SetAppliedDirenv/ClearAppliedDirenv` 与其测试。
  3. 保留 `AppliedDirenv`/`AppliedEnv` 结构与其自身测试（结构仍被 T2 使用）。
- 验证: `go test ./internal/xenv/models/ ./internal/xenv/manager/ -count=1`；全仓 grep 确认无残留引用。
- 完成标准: session state 不再承载记录，无未使用 API（staticcheck 通过）。
- 依赖: T2。

### T3 SetupDirenv 拆分：撤销 + 应用 + 记录（环境变量版）

- 文件: `internal/xenv/service/sdk_service.go`（改动）、`internal/xenv/service/tool_service_test.go`（改动）
- 动作:
  1. `SetupDirenv` 用 `loadAppliedRecord()` 读取记录：无记录或记录 `file` 与即将应用的 `.xenv.toml` 相同 → 跳过撤销。
  2. 否则生成撤销脚本：用 `GenSetPath` 移除记录中的 PATH 条目（复用 `sessionPath`/`withoutPath`），对 `Envs` 逐项 `GenSetEnv(prev)` 或 `GenUnsetEnv(name)`，并在末尾清除记录变量。
  3. 应用当前目录（0.1 checkpoint 的 `applyDirenvState` 逻辑不变），计算 delta 后由 `writeAppliedRecord` 写入脚本末尾。
  4. 撤销失败（无条目可移除等）只 WARN，不返回错误。
- 验证: `go test ./internal/xenv/service/ -count=1`；新增用例：首次进入写记录、同目录幂等、跨目录撤销+应用、无记录变量、损坏记录变量。
- 完成标准: 验收 1-5 有单测证据；非 hook shell（`gen == nil`）行为不变。
- 依赖: T2、T2b。

### T4 `-s` 系列命令合并记录

- 文件: `internal/xenv/service/env_service.go`（改动）、`internal/xenv/service/sdk_service.go`（改动，如需）、`internal/xenv/service/tool_service_test.go`（改动）
- 动作:
  1. 抽出单一 helper（如 `recordAppliedDelta(gen, dirFile, paths, envs, sdks)`）：读取当前记录、合并本次 delta（同名变量保留最早旧值），返回写入记录的脚本行。
  2. 在 `SetEnvs`/`UnsetEnvs`/`AddPath`/`RemovePath`/`RemoveMatchedPaths` 的 direnv 分支与 `ActivateSDKs`（direnv）路径中，于生成脚本后调用该 helper；非 hook shell 不写。
  3. `UnsetEnvs` 的 direnv 分支在记录中移除对应变量（离开时无需再恢复）。
- 验证: `go test ./internal/xenv/service/ -count=1`；新增用例：`set -s` 后记录包含该变量及旧值、`path add -s` 后记录包含该路径、`unset -s` 后记录中移除该变量。
- 完成标准: hook shell 内 direnv 作用域命令的变更都能被离开目录时撤销。
- 依赖: T3。

### T5 展示与 `dir_states` 收敛

- 文件: `internal/cli/status_cmd.go`（改动）、`internal/cli/status_cmd_test.go`（改动）
- 动作:
  1. `--layers` 的 Directory State 段改为展示记录变量内容（文件、paths、envs、sdks）；无记录时显示 “No applied direnv state found”。
  2. 展示读取与撤销读取共用同一解析函数（避免两套解析）。
- 验证: `go test ./internal/cli/ -count=1`；实测 `xenv status --layers` 输出。
- 完成标准: 验收 7 满足；展示与撤销读取同一份数据。
- 依赖: T2、T3。

### T6 文档

- 文件: `README.md`、`README.zh-CN.md`
- 动作: 在 Project State 一节补充「离开目录会撤销该目录带来的 PATH/ENV/SDK 变更」、不可撤销项（项目脚本/`.envrc` 副作用）、幂等与失败行为，以及记录变量 `XENV_APPLIED_DIRENV` 的存在（便于排查）。
- 验证: 人工核对示例与实际行为一致。
- 完成标准: 双语文档均说明离开行为与限制。
- 依赖: T3。

### T7 端到端验证

- 文件: 无（临时脚本放 `tmp/`，验证后删除）
- 动作:
  1. 构造临时 HOME + `XENV_CONFIG_DIR` + 两个含 `.xenv.toml` 的项目目录（含 paths/envs/sdks）与一个无配置目录。
  2. 模拟 hook：以 `XENV_HOOK_SHELL=pwsh` 运行 `xenv init-direnv`，把输出的 `--Expression--` 之后脚本在**同一 shell 环境**中 eval（可用 pwsh 执行），核对：首次应用并写入记录变量、同目录幂等、跨目录撤销、离开到无配置目录、记录变量被清除。
  3. 记录每条命令输出、`PATH`/变量值与记录变量内容作为证据。
- 验证: 上述实测输出与验收 1-6 一致。
- 完成标准: 验收 1-6 全部可复现；`tmp/` 清理完毕。
- 依赖: T3、T4、T5。

## 回滚与恢复

- 每个任务一个本地 atomic commit，提交前核对 `git diff --cached --name-only` 与本任务 owner 文件一致。
- 回滚：`git revert <commit>`；或移除记录变量的读写与撤销步骤（记录变量对其它功能无影响，仅 `status` 读取）。
- 恢复点：W1 结束（记录可经环境变量往返）、W2 结束（撤销/应用/记录可用）、W3 结束（验收通过）。
- dirty 保护：任一阶段发现 owner 外修改即停止并按 Ownership Conflict 处理。

## 人工 Gate

| Gate | 内容 | 状态 |
|---|---|---|
| G1 计划批准 | 用户批准本计划后进入实施 | 0.1 已批准；0.2 修订为方案 A 的直接后果 |
| G2 当前执行请求 | 批准计划不自动授权执行；需单独“开始实施”请求 | 0.1 已给出，0.2 修订范围内继续 |
| 外部动作 Gate | 不适用（`host_or_non_offline_action=NOT_APPLICABLE`） | 不适用 |

## 可追溯性

| 设计条目 | 任务 | 验证 |
|---|---|---|
| 验收 1（离开撤销 PATH/ENV/SDK） | T2、T3、T7 | 单测 + T7 实测 |
| 验收 2（同项目幂等） | T3、T7 | 单测 + T7 实测 |
| 验收 3（A→B 撤销再应用） | T3、T7 | 单测 + T7 实测 |
| 验收 4（无记录变量兼容） | T2、T3、T7 | 单测 + T7 实测 |
| 验收 5（损坏记录只 WARN） | T2、T3 | 单测 |
| 验收 6（非 hook 行为不变） | T3 | 单测 |
| 验收 7（status 展示） | T5 | 单测 + 实测 |
| 验收 8（测试全绿） | T1-T5 | `go test -count=1 ./...` |
| D1（命令面/模板不变） | T3 | 无模板改动，`git diff` 核对 |
| D2（delta 记录） | T1、T3 | T1/T3 单测 |
| D3（记录经环境变量承载） | T2、T2b、T3 | T2/T3 单测 + 全仓 grep 无 session 承载残留 |
| D4（同项目幂等） | T3 | T3 单测 |
| D5（`-s` 命令记录） | T4 | T4 单测 |
| D6（失败不阻断） | T3 | T3 单测 |
| D7（变量恢复/取消设置） | T1、T3 | T1/T3 单测 |
| D8（收敛 dir_states 展示） | T5 | T5 单测 + 实测 |
| D9（JSON 文本 + 常量集中） | T2 | T2 单测 |

## 完成 Gate 与剩余工作

完成 Gate（全部满足才可声明完成）：

1. 验收 1-8 均有实测或测试证据。
2. `go build ./...`、`go vet ./...`、`staticcheck ./...` 通过；`go test -count=1 ./...` 全绿。
3. 消费本计划声明的 `host_or_non_offline_action=NOT_APPLICABLE`；不产生外部动作。
4. 临时产物（`tmp/` 下的验证脚本与临时 HOME）已清理。

剩余工作（不在本次范围）：

- Q2 的手动 `leave-direnv` 命令、Q4 的脚本副作用隔离（子 shell 方案）若日后需要，按 `Semantic Amendment` 返回设计/计划修订与人工 Gate。
- 多 `.xenv.toml` 叠加、非 hook shell 的离开逻辑、session 文件按目录键控的既有语义均未列入任何波次。
