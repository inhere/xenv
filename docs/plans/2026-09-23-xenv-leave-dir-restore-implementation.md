<!-- template_id: plan; template_version: 1.2.0 -->
# xenv 离开目录恢复会话默认值实施计划

> 状态：Draft 0.1 / 待人工计划批准

## 修订记录

| 版本 | 日期 | 作者 | 摘要 |
|---|---|---|---|
| 0.1 | 2026-09-23 | Pi | 初稿，按已批准设计 0.1 拆分波次与任务，记录 workspace baseline、验证方式与人工 Gate。 |

> 仅语义变化递增版本；纯 identity/provenance/元数据纠正沿用原版本，并在 Git/进度记录中留痕。

## 目标与完成定义

目标：按已批准设计 `docs/design/2026-09-23-xenv-leave-dir-restore-design.md`（0.1）实现「离开目录撤销 direnv 变更」：hook 仍在 cd 后调用 `xenv init-direnv`，由命令内部先撤销上一次应用、再应用当前目录，并维护 session 中的应用记录。

完成定义（可观察）：

1. hook shell 中进入含 `.xenv.toml`（paths/envs/sdks）的目录后，PATH/ENV/SDK 生效；`cd` 到无配置目录后，该目录新增的 PATH 条目消失、受影响变量恢复进入前的值（原先未设置的变为未设置）。
2. 进入项目目录后再进入其子目录（同一 `.xenv.toml`）不重复应用，PATH 不出现重复条目。
3. 项目 A → 项目 B：A 的变更被撤销、B 生效。
4. 旧 session 文件（无 `applied_direnv`）不报错，首次进入正常应用。
5. 撤销失败（如记录中的条目已不在 PATH）不阻断 cd，只输出 WARN。
6. 非 hook shell 执行 `xenv init-direnv` 行为与当前一致（只应用、不撤销）。
7. `xenv status --layers` 展示当前应用记录（替代原先读取 `dir_states` 的展示）。
8. `go test ./...` 全绿；新增单测覆盖记录读写、撤销、幂等与兼容。

## 范围、排除项与授权

范围：设计 0.1 的范围条目 1-5 与决策 D1-D8。

排除项：设计中的非目标 1-5；待确认事项 Q1-Q5 按默认值处理（Q1 恢复进入前的值、Q2 不新增 `leave-direnv`、Q3 仅在 `--layers` 展示、Q4 接受脚本副作用不可撤销、Q5 记录放 session state）。

授权边界：仅本地代码改动、本地测试与本地 atomic commit（`SR1211`）；不 push/tag/release/部署；不改 `.xenv.toml` 与 global state 的既有语义；不修改四个 hook 模板。

- `host_or_non_offline_action=NOT_APPLICABLE`

## 输入与批准证据

- 设计：`docs/design/2026-09-23-xenv-leave-dir-restore-design.md`（0.1）。
- 设计 candidate：`git_root=D:\work\inhere\my-tools-dev\inhere-tools\xenv`、`commit=791d896d09c23dc2464569c47bacef41a7c6b4ba`、`subject_path=docs/design/2026-09-23-xenv-leave-dir-restore-design.md`、`document_revision=0.1`。
- 设计批准证据：用户 2026-09-23 会话原话“批准 docs/design/2026-09-23-xenv-leave-dir-restore-design.md 的设计；待确认事项 Q1-Q5 中未确认的按设计默认值处理。”
- 规范：`SR1204`、`SR1207`、`SR1403`、`SR1407`、`SR1107`、`SR1206`、`SR1211`、`GR104`、`GR106`。

## 工作区基线

| 项 | 值 |
|---|---|
| Git root | `D:\work\inhere\my-tools-dev\inhere-tools\xenv` |
| Branch | `main` |
| HEAD | `791d896d09c23dc2464569c47bacef41a7c6b4ba` |
| `git status --short` | 空（无 in-scope 或 unrelated dirty/untracked） |
| 保留的无关 dirty 路径 | 无 |
| 依赖与能力 | Go 1.25.10（`go.mod` 要求 1.24）；无新增第三方依赖；端到端验证需要已安装 SDK（`xenv sdk list` 可见 go）与可用的 hook shell |
| 授权与生命周期边界 | 本地实现 + 本地验证 + 本地 atomic commit；无出机器动作 |
| Expected paths | 见各任务“文件”；Expected symbols：`models.AppliedDirenv`、`models.AppliedEnv`、`StateManager.AppliedDirenv`、`StateManager.SetAppliedDirenv`、`StateManager.ClearAppliedDirenv`、`SDKService.SetupDirenv`、`EnvService.recordAppliedDelta` |

## Capability Discovery

### Capability decisions

| capability_id | required_capability | searched_candidates | direct_reuse | thin_adapter_or_owner_extension | decision | proven_gap | duplication_and_lifecycle_risk |
|---|---|---|---|---|---|---|---|
| CAP-01 | 应用记录的结构与 session 持久化 | `models/state.go: ActivityState/DirStates/AddDirState`；`manager/state_manager.go: LoadStateFiles/SaveStateFile/saveStateFile`；`goutil/jsonutil` 写入 | `ActivityState` 字段 + 既有 JSON 持久化路径 | 在 `ActivityState` 增加 `AppliedDirenv` 字段与增删方法；`StateManager` 增加读写方法 | OWNER_EXTENSION | none | 与 `dir_states` 并存会产生两套记录，必须按 D8 收敛展示与撤销职责 |
| CAP-02 | 撤销脚本生成（PATH 移除、变量恢复/取消设置） | `shell.XenvScriptGenerator: GenSetPath/GenSetEnv/GenUnsetEnv`；`util.SplitPath/JoinPaths`；`service: sessionPath/withoutPath` | 直接复用既有生成器与 PATH 工具函数 | 无 | DIRECT_REUSE | none | 不得另写一套引号/转义或 PATH 比较逻辑（本会话刚统一过） |
| CAP-03 | 应用/撤销编排与记录写入 | `service/sdk_service.go: SetupDirenv`；`service/env_service.go: SetEnvs/UnsetEnvs/AddPath/RemovePath`；`manager/state_manager.go` direnv 分支 | 既有 direnv 应用逻辑 | `SetupDirenv` 拆为撤销/应用/记录三步；新增单一记录 helper 供 `-s` 命令复用 | OWNER_EXTENSION | none | 记录写入若分散在多个命令里会漏记，收敛到 helper |
| CAP-04 | 展示当前应用记录 | `cli/status_cmd.go: buildEffectiveSDKRows/formatLayerLines/DirStates()` | 既有展示函数与分层输出 | `--layers` 改为读取应用记录 | THIN_ADAPTER | 现有展示读取 `dir_states`，语义与撤销记录不同 | 展示与撤销必须读同一份数据，避免用户看到不一致 |
| CAP-05 | 测试支撑（hook shell、session 文件断言） | `service` 包的 `newDirenvTestService`；`manager` 包状态测试；`cli` 包纯函数测试 | 直接复用既有 helper | 无 | DIRECT_REUSE | none | 新测试不得依赖 ambient PATH/HOME（本会话已修过同类问题） |

### New module candidates

| candidate_id | capability_id | proposed_module | searched_candidates | direct_reuse_gap | thin_adapter_or_owner_extension_gap | proven_gap | unique_owner_and_lifecycle | deletion_or_merge_handling |
|---|---|---|---|---|---|---|---|---|
| NONE | none | none | none | none | none | none | none | none |

### Rejected new tools

| rejected_candidate | capability_id | deletion_test_and_reason |
|---|---|---|
| 整环境快照回滚（保存完整 PATH/ENV 并在离开时整体还原） | CAP-03 | 删除测试：delta 记录已覆盖设计验收 1-3 的全部可撤销项；快照方案需要 hook 额外保存完整环境，状态体积与失败面更大 |
| 新增 `leave-direnv` 命令并在四个 hook 模板中调用 | CAP-03 | 删除测试：由 `init-direnv` 内部撤销即可满足验收，且不必改 4 个模板、不增加 hook 往返 |
| 独立的应用记录文件（如 `session/<id>.direnv.json`） | CAP-01 | 删除测试：session state 已是 hook shell 内的既有持久化载体，独立文件增加一次 IO 与清理路径 |

## 前置检查与 fail-closed 条件

前置检查：

1. `git status --short` 为空且 HEAD 为 `791d896`；否则先确认 dirty 归属。
2. `go build ./...`、`go vet ./...`、`staticcheck ./...` 通过。
3. 设计 0.1 已批准且 candidate 与上表一致。
4. 端到端验证需要：已安装的 go SDK（`xenv sdk list` 可见）、可在 Windows 上构造临时 HOME 与 hook 环境。

fail-closed 条件（命中即停止并回到 design/plan Gate）：

- 需要修改四个 hook 模板或新增 hook 命令才能满足验收（与 D1 冲突）。
- 需要改变 `.xenv.toml`、global state 或 `-s` 的持久化语义。
- 需要引入新依赖或新包。
- 命中他人在途修改（Ownership Conflict）。

## 波次与依赖

| 波次 | 内容 | 任务 | 依赖 |
|---|---|---|---|
| W1 | 记录结构与持久化 | T1、T2 | 设计 0.1 |
| W2 | 撤销与应用编排 | T3、T4 | W1 |
| W3 | 展示、文档与端到端验证 | T5、T6、T7 | W2 |

## 任务

### T1 应用记录结构

- 文件: `internal/xenv/models/state.go`（改动）
- 动作:
  1. 新增 `AppliedDirenv{File string; Paths []string; Envs []AppliedEnv; SDKs map[string]string; AppliedAt time.Time}` 与 `AppliedEnv{Name, Prev string; HadPrev bool}`，JSON tag 使用 `applied_direnv`/`had_prev`。
  2. 在 `ActivityState` 增加 `AppliedDirenv *AppliedDirenv` 字段（`json:"applied_direnv,omitempty" toml:"-"`）。
  3. 增加方法：`SetAppliedDirenv(*AppliedDirenv)`（置 `HasUpdate`）、`ClearAppliedDirenv()`（置 `HasUpdate` 并清空）、`HasAppliedDirenv() bool`。
- 验证: `go build ./... && go vet ./...`；`go test ./internal/xenv/models/ -count=1`。
- 完成标准: 结构可通过既有 JSON 持久化路径序列化/反序列化；`IsEmpty()` 行为不受影响（记录不算激活内容）。
- 依赖: 无（W1 首个任务）。

### T2 记录的读写与持久化

- 文件: `internal/xenv/manager/state_manager.go`（改动）、`internal/xenv/manager/state_manager_test.go`（改动）
- 动作:
  1. 增加 `AppliedDirenv() *models.AppliedDirenv`、`SetAppliedDirenv(rec *models.AppliedDirenv) error`、`ClearAppliedDirenv() error`，写操作走既有 `SaveStateFile()`（session 仅在 hook shell 保存，保持现状）。
  2. 确认 `LoadStateFiles` 读取旧文件时缺字段不报错。
- 验证: `go test ./internal/xenv/manager/ -count=1`；新增用例覆盖「写入→重新加载→读回」与「旧文件无字段→返回 nil」。
- 完成标准: 记录在 session 文件中持久化，旧文件兼容；不改变 `dir_states` 的既有行为（其清理在 T5 处理）。
- 依赖: T1。

### T3 SetupDirenv 拆分：撤销 + 应用 + 记录

- 文件: `internal/xenv/service/sdk_service.go`（改动）、`internal/xenv/service/tool_service_test.go`（改动）
- 动作:
  1. `SetupDirenv` 开头读取应用记录：无记录或记录 `file` 与即将应用的 `.xenv.toml` 相同 → 跳过撤销。
  2. 否则生成撤销脚本：用 `GenSetPath` 移除记录中的 PATH 条目（复用 `sessionPath`/`withoutPath` 比较规则），对 `Envs` 逐项 `GenSetEnv(prev)` 或 `GenUnsetEnv(name)`。
  3. 应用当前目录（现有逻辑不变），并按「应用前进程环境」计算 delta：新加入 PATH 的条目、被设置的变量名及其旧值（`os.Getenv` 为空视为未设置）、激活的 SDK。
  4. 写入新记录；撤销脚本与应用脚本按顺序拼接在同一次返回值中。
  5. 撤销失败（无条目可移除等）只 WARN，不返回错误。
- 验证: `go test ./internal/xenv/service/ -count=1`；新增用例：首次进入写记录、同目录幂等、跨目录撤销+应用、旧 session 无记录、撤销时条目已不存在。
- 完成标准: 验收 1-3、5 有单测证据；`SetupDirenv` 在非 hook shell（`gen == nil`）行为不变。
- 依赖: T2。

### T4 `-s` 系列命令同步记录

- 文件: `internal/xenv/service/env_service.go`（改动）、`internal/xenv/service/sdk_service.go`（改动，如需）、`internal/xenv/service/tool_service_test.go`（改动）
- 动作:
  1. 抽出单一 helper（如 `recordAppliedDelta(dirFile string, paths []string, envs map[string]string, sdks map[string]string)`），内部读取当前记录并合并 delta（同名变量保留最早一次 `prev`）。
  2. 在 `SetEnvs`/`UnsetEnvs`/`AddPath`/`RemovePath`/`RemoveMatchedPaths` 的 direnv 分支、以及 `ActivateSDKs`（direnv）路径中，于生成脚本后调用该 helper；非 hook shell 不写。
  3. `UnsetEnvs` 的 direnv 分支在撤销记录中同样要移除对应变量（离开目录时无需再恢复）。
- 验证: `go test ./internal/xenv/service/ -count=1`；新增用例：`set -s` 后记录包含该变量及旧值、`path add -s` 后记录包含该路径、`unset -s` 后记录中移除该变量。
- 完成标准: hook shell 内 direnv 作用域命令的变更都能被离开目录时撤销。
- 依赖: T3（共用 helper）。

### T5 展示与 `dir_states` 收敛

- 文件: `internal/cli/status_cmd.go`（改动）、`internal/cli/status_cmd_test.go`（改动）、`internal/xenv/manager/state_manager.go`（改动，移除 `dir_states` 展示依赖时按需）
- 动作:
  1. `--layers` 的 Directory State 段改为展示当前应用记录（文件、paths、envs、sdks）。
  2. 无记录时显示 “No applied direnv state found”。
  3. 确认 `dir_states` 不再承担撤销职责；若移除其写入/展示，同步更新受影响测试。
- 验证: `go test ./internal/cli/ -count=1`；实测 `xenv status --layers` 输出。
- 完成标准: 验收 7 满足；展示与撤销读取同一份数据。
- 依赖: T2、T3。

### T6 文档

- 文件: `README.md`、`README.zh-CN.md`
- 动作: 在 Project State 一节补充「离开目录会撤销该目录带来的 PATH/ENV/SDK 变更」、不可撤销项（项目脚本/`.envrc` 副作用）、以及幂等与失败行为。
- 验证: 人工核对示例与实际行为一致。
- 完成标准: 双语文档均说明离开行为与限制。
- 依赖: T3。

### T7 端到端验证

- 文件: 无（临时脚本放 `tmp/`，验证后删除）
- 动作:
  1. 构造临时 HOME + `XENV_CONFIG_DIR` + 两个含 `.xenv.toml` 的项目目录（含 paths/envs/sdks）与一个无配置目录。
  2. 用 `XENV_HOOK_SHELL=pwsh` 与真实二进制依次执行 `init-direnv` 并 eval 输出（可用 pwsh 或直接检查脚本内容与 session 文件），核对：首次应用、同目录幂等、跨目录撤销、离开到无配置目录、旧 session 兼容。
  3. 记录每条命令输出与 session 文件内容作为证据。
- 验证: 上述实测输出与验收 1-6 一致。
- 完成标准: 验收 1-6 全部可复现；`tmp/` 清理完毕。
- 依赖: T3、T4、T5。

## 回滚与恢复

- 每个任务一个本地 atomic commit，提交前核对 `git diff --cached --name-only` 与本任务 owner 文件一致。
- 回滚：`git revert <commit>`；或删除 `AppliedDirenv` 字段与撤销步骤（记录字段对其它功能无影响，仅 `status` 读取）。
- 恢复点：W1 结束（记录可持久化）、W2 结束（撤销/应用/记录可用）、W3 结束（验收通过）。
- dirty 保护：任一阶段发现 owner 外修改即停止并按 Ownership Conflict 处理。

## 人工 Gate

| Gate | 内容 | 状态 |
|---|---|---|
| G1 计划批准 | 用户批准本计划后进入实施 | 待批准 |
| G2 当前执行请求 | 批准计划不自动授权执行；需单独“开始实施”请求 | 未触发 |
| 外部动作 Gate | 不适用（`host_or_non_offline_action=NOT_APPLICABLE`） | 不适用 |

## 可追溯性

| 设计条目 | 任务 | 验证 |
|---|---|---|
| 验收 1（离开撤销 PATH/ENV/SDK） | T2、T3、T7 | 单测 + T7 实测 |
| 验收 2（同项目幂等） | T3、T7 | 单测 + T7 实测 |
| 验收 3（A→B 撤销再应用） | T3、T7 | 单测 + T7 实测 |
| 验收 4（旧 session 兼容） | T2、T7 | 单测 + T7 实测 |
| 验收 5（失败只 WARN） | T3 | 单测 |
| 验收 6（非 hook 行为不变） | T3 | 单测 |
| 验收 7（status 展示） | T5 | 单测 + 实测 |
| 验收 8（测试全绿） | T1-T5 | `go test -count=1 ./...` |
| D1（命令面/模板不变） | T3 | 无模板改动，`git diff` 核对 |
| D2（delta 记录） | T1、T3 | T1/T3 单测 |
| D3（记录入 session state） | T1、T2 | T2 单测 |
| D4（同项目幂等） | T3 | T3 单测 |
| D5（`-s` 命令记录） | T4 | T4 单测 |
| D6（失败不阻断） | T3 | T3 单测 |
| D7（变量恢复/取消设置） | T1、T3 | T1/T3 单测 |
| D8（收敛 dir_states） | T5 | T5 单测 + 实测 |

## 完成 Gate 与剩余工作

完成 Gate（全部满足才可声明完成）：

1. 验收 1-8 均有实测或测试证据。
2. `go build ./...`、`go vet ./...`、`staticcheck ./...` 通过；`go test -count=1 ./...` 全绿。
3. 消费本计划声明的 `host_or_non_offline_action=NOT_APPLICABLE`；不产生外部动作。
4. 临时产物（`tmp/` 下的验证脚本与临时 HOME）已清理。

剩余工作（不在本次范围）：

- Q2 的手动 `leave-direnv` 命令、Q4 的脚本副作用隔离（子 shell 方案）若日后需要，按 `Semantic Amendment` 返回设计/计划修订与人工 Gate。
- 多 `.xenv.toml` 叠加、非 hook shell 的离开逻辑均未列入任何波次。
