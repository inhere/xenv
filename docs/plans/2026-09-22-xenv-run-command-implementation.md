<!-- template_id: plan; template_version: 1.2.0 -->
# xenv run 一次性环境执行命令实施计划

> 状态：Draft 0.1 / 待人工计划批准

## 修订记录

| 版本 | 日期 | 作者 | 摘要 |
|---|---|---|---|
| 0.1 | 2026-09-22 | Pi | 初稿，按已批准设计 0.2 拆分波次与任务，记录 workspace baseline、验证方式与人工 Gate。 |

> 仅语义变化递增版本；纯 identity/provenance/元数据纠正沿用原版本，并在 Git/进度记录中留痕。

## 目标与完成定义

目标：按已批准设计 `docs/design/2026-09-22-xenv-run-command-design.md`（0.2）交付 `xenv run`（别名 `exec`）：用 `--use/--path/--env/--cwd/--print` 一次性合成环境并执行目标命令，不读写任何 xenv 状态、不依赖 shell hook、透传子进程退出码。

完成定义（可观察）：

1. `xenv run -u go:1.25 -e APP_ENV=local --cwd <dir> -- go version` 输出该 SDK 版本，且子进程工作目录为 `<dir>`。
2. 退出码透传：`xenv run -- cmd /c exit 3`（Windows）/ `xenv run -- sh -c 'exit 3'`（Unix）返回 3；命令不存在返回 127。
3. 执行前后 `~/.config/xenv/global.toml`、`session/*.json`、当前目录 `.xenv.toml` 内容与 mtime 均不变。
4. 未定义 SDK、未安装版本、`--path` 目录不存在、`--cwd` 非目录、`--env` 缺少 `=` 均报错且不启动子进程。
5. `--print` 打印增量环境、最终 PATH 与将执行的命令，且不执行命令。
6. 新增单测通过：`go test ./internal/xenv/service/ -run 'Run'`。
7. `README.md` 与 `README.zh-CN.md` 含命令用法与示例。

## 范围、排除项与授权

范围：设计 0.2 的范围条目 1-4 与决策 D1-D10。

排除项：设计中的待确认事项 Q1、Q3-Q7（未确认即不实现）；非目标 1-5（不写状态、不改既有命令语义、不做下载安装、不做交互式子 shell/信号转发、不做并发与 JSON 输出）。

授权边界：仅本地代码改动、本地测试与本地 atomic commit（`SR1211`）；不 push/tag/release/部署；不改动 `-S/--system` 相关行为；不修改用户真实 xenv 状态文件与系统环境。

- `host_or_non_offline_action=NOT_APPLICABLE`

## 输入与批准证据

- 设计：`docs/design/2026-09-22-xenv-run-command-design.md`（0.2）。
- 设计 candidate：`git_root=D:\work\inhere\my-tools-dev\inhere-tools\xenv`、`commit=e2d75211619741492e363006432b8d670c82b8d4`、`subject_path=docs/design/2026-09-22-xenv-run-command-design.md`、`document_revision=0.2`。
- 设计批准证据：用户 2026-09-22 会话原话“加上 --cwd 支持；批准设计”。
- 规范：`SR1204`（范围确认）、`SR1207`（流程）、`SR1403`/`SR1405`（最小实现）、`SR1407`（最小验证）、`SR1206`（提交前缀）、`SR1211`（本地原子提交）、`GR104`（文件规模）、`GR106`（测试断言）。

## 工作区基线

| 项 | 值 |
|---|---|
| Git root | `D:\work\inhere\my-tools-dev\inhere-tools\xenv` |
| Branch | `main` |
| HEAD | `e2d75211619741492e363006432b8d670c82b8d4` |
| `git status --short` | 空（无 in-scope 或 unrelated dirty/untracked） |
| 保留的无关 dirty 路径 | 无 |
| 依赖与能力 | Go 1.25.10（`go.mod` 要求 1.24）；无新增第三方依赖；需要本机已安装 go/node 用于端到端验证 |
| 授权与生命周期边界 | 本地实现 + 本地验证 + 本地 atomic commit；无出机器动作 |
| Expected paths | 见各任务“文件”；Expected symbols：`service.RunService`、`service.RunOptions`、`cli.NewRunCmd`、`xenv.RunService`、`sysenv.NormalizeWinPath` |

## Capability Discovery

### Capability decisions

| capability_id | required_capability | searched_candidates | direct_reuse | thin_adapter_or_owner_extension | decision | proven_gap | duplication_and_lifecycle_risk |
|---|---|---|---|---|---|---|---|
| CAP-01 | SDK 规格解析与本地安装解析（name:version → 安装信息） | `internal/xenv/sdk/version.go: ParseVersionSpec/ParseMultipleVersionSpecs`；`internal/xenv/service/sdk_service.go: checkActivateSDK/WhereSDK`；`internal/xenv/manager/sdk_manager.go: ListSDKVersions/MatchSDKByVersion/ListMergedSDKVersions` | `sdk.ParseVersionSpec` + 同包 `SDKService.checkActivateSDK` 直接复用 | 无 | DIRECT_REUSE | none | 不复制版本匹配逻辑，避免与 `use` 命令的匹配行为漂移 |
| CAP-02 | 环境与 PATH 合成（覆盖顺序、去重、平台分隔符） | `models.NewActivateSDKsParams`（AddPath/AddSetEnvs，面向脚本与状态写入）；`util.SplitPath/JoinPaths`；`xenvcom.PathSep`；`sysenv/pathlist.go: normalizeWinPath`；`shell.XenvScriptGenerator`（只产出脚本） | `util.SplitPath`/`util.JoinPaths`/`xenvcom.PathSep`；导出并复用 `sysenv.NormalizeWinPath` | service 内新增纯函数 `composeEnv`/`composePath`（约 60 行） | THIN_ADAPTER | 既有聚合类型只服务 shell 脚本与状态写入，不产出“继承环境 + 覆盖 + 去重”的进程环境 | 去重比较必须复用 sysenv 的 Windows 归一化规则，避免出现第二套路径比较语义 |
| CAP-03 | 在合成 PATH 中解析可执行文件并执行子进程、透传退出码 | stdlib `os/exec`（Command/LookPath）；`goutil/sysutil`（无 exec 封装）；仓库内无执行封装 | `os/exec` + `exec.LookPath` | service 内约 30 行：临时设置进程 PATH → `LookPath` → 立即恢复 | THIN_ADAPTER | `exec.Command` 用父进程 PATH 解析，需显式桥接合成 PATH | 临时修改进程 PATH 的副作用必须限定在解析阶段并恢复，避免影响同进程后续逻辑 |
| CAP-04 | CLI 命令、可重复选项与 `--` 之后的参数透传 | `internal/cli/app.go` 与既有 `*_cmd.go` 模式；gcli `gflag` 选项 API（`VarOpt`/`flag.Value`、`AddArg`、`--` 终止解析）；`errorx.Failf` | 既有命令骨架与 gcli 选项绑定 | `run_cmd.go` 内约 120 行：可重复选项值类型、`--` 切分、退出码落地 | THIN_ADAPTER | 既有命令无“透传子进程退出码”需求；gcli 错误通道会加 `ERROR:` 前缀且普通 error 返回 0 | 选项短名需与 app 级 `-d/--debug` 等避开；不得全局关闭 gcli 参数重排 |
| CAP-05 | 指定子进程工作目录 | stdlib `exec.Cmd.Dir` | `exec.Cmd.Dir` 直接复用 | 无 | DIRECT_REUSE | none | 不做 `os.Chdir`，避免全局副作用 |
| CAP-06 | 服务装配入口 | `internal/xenv/xenv.go: EnvService()/SDKService()` 模式 | 同文件新增 `RunService()` 构造函数 | 既有 owner 文件扩展装配职责 | OWNER_EXTENSION | none | `RunService` 不得持有 `StateManager`，从构造上排除写状态 |

### New module candidates

| candidate_id | capability_id | proposed_module | searched_candidates | direct_reuse_gap | thin_adapter_or_owner_extension_gap | proven_gap | unique_owner_and_lifecycle | deletion_or_merge_handling |
|---|---|---|---|---|---|---|---|---|
| NONE | none | none | none | none | none | none | none | none |

### Rejected new tools

| rejected_candidate | capability_id | deletion_test_and_reason |
|---|---|---|
| 自实现 `lookPathIn(dirs, name)` 查找器 | CAP-03 | 删除后仍可用 `exec.LookPath` + 临时 PATH 满足同一验收；保留会复制 `PATHEXT`/可执行位等平台规则，形成第二套解析实现 |
| 新增 `internal/xenv/runenv` 包 | CAP-02 | 删除后合成逻辑仍在既有 `service` 包内被单一调用方使用；新包只服务单点用途，却引入 Module 生命周期与命名成本 |
| 全局关闭 gcli 参数重排 | CAP-04 | 删除后 `--` 分隔即可满足验收；全局改动会改变 `use/env/path` 等既有命令的选项解析行为 |

## 前置检查与 fail-closed 条件

前置检查：

1. `git status --short` 为空，且 HEAD 为 `e2d7521`；否则先确认 dirty 归属。
2. `go build ./...` 与 `go vet ./...` 通过。
3. 设计 0.2 已批准且 candidate 与上表一致。
4. 端到端验证需要本机存在可用的 SDK 安装（`xenv sdk list` 可见 go），否则该任务改为记录跳过原因。

fail-closed 条件（命中即停止并回到 design/plan Gate）：

- `checkActivateSDK` 的解析语义与设计假设不符，或需要改动其现有行为。
- gcli 行为与已验证的机制不符：`c.AddArg` 后位置参数不再触发“subcommand is not found”、`flag.Value` 可重复选项、`--` 之后参数原样保留（本计划已用 `tmp/gcliprobe` 探针验证，探针已删除）。
- 需要写入任何 xenv 状态文件或系统环境才能完成验收。
- 命中他人在途修改（Ownership Conflict）。

## 波次与依赖

| 波次 | 内容 | 任务 | 依赖 |
|---|---|---|---|
| W1 | 服务层合成与执行内核 | T1、T2 | 设计 0.2 |
| W2 | CLI 接线与退出码 | T3 | W1 |
| W3 | 端到端验证与文档 | T4、T5 | W2 |

每个任务完成后创建一个本地 atomic commit；W2 依赖 W1 的 `RunService` 签名，W3 依赖 W2 的可执行命令。

## 任务

### T1 实现 RunService 合成与执行内核

- 文件: `internal/xenv/service/run_service.go`（新增）、`internal/xenv/sysenv/pathlist.go`（改动：`normalizeWinPath` 重命名为导出 `NormalizeWinPath`，行为不变，同步其包内调用点）
- 动作:
  1. 定义 `RunOptions{Use []string; Paths []string; Envs []string; Cwd string}` 与 `RunService`（持有 `*models.Configuration`、`*manager.SDKManager`、`*SDKService`，不持有 `StateManager`）。
  2. `ResolveSDKs(specs []string) ([]*models.InstalledSDK, error)`：`sdk.ParseVersionSpec` → `sdkSvc.checkActivateSDK`，错误原样上抛。
  3. `composeEnv(base []string, sdkEnvs map[string]string, cliEnvs []string) ([]string, error)`：解析 `KEY=VALUE`、键名大写并用 `strutil.IsVarName` 校验，后者覆盖前者，保留 base 中未涉及的变量。
  4. `composePath(cliPaths, sdkBins []string, basePath string) []string`：`util.SplitPath` 拆分，Windows 用 `sysenv.NormalizeWinPath` 归一化后 `EqualFold` 去重，其它平台按原样去重；丢弃空条目；顺序为 `--path` → SDK bin → 继承 PATH。
  5. `BuildEnv(opts RunOptions) (*RunEnv, error)`：校验 `--cwd` 为已存在目录；返回 `Env []string`、`Path string`、`Added []string`（供 `--print` 打印增量）。
  6. `Run(env *RunEnv, command []string) (int, error)`：临时 `os.Setenv("PATH", env.Path)` → `exec.LookPath` → `defer` 恢复原 PATH；命令缺失返回 `127` 与 `command not found: <name>`；否则 `exec.Cmd{Dir, Env, Stdin/out/err 继承}` 执行并返回退出码。
- 验证: `go build ./... && go vet ./...`；`go test ./internal/xenv/sysenv/`（确认重命名未破坏既有测试）。
- 完成标准: 上述符号可按签名调用；`RunService` 不引用 `StateManager`；无编译告警。
- 依赖: 无（W1 首个任务）。

### T2 合成规则与解析错误单测

- 文件: `internal/xenv/service/run_service_test.go`（新增）
- 动作:
  1. 用 `manager.NewSDKManager(临时索引文件)` + `AddSDK` 构造最小本地索引，不触碰用户真实索引。
  2. 覆盖 `composeEnv`：覆盖顺序、非法键名报错、`KEY=` 空值、base 未涉及变量保留。
  3. 覆盖 `composePath`：顺序、重复条目去重（Windows 大小写/分隔符差异）、空条目丢弃。
  4. 覆盖 `BuildEnv`：`--cwd` 非目录报错、`--path` 不存在报错、未定义 SDK 与未安装版本报错。
  5. 断言统一使用 `github.com/gookit/goutil/x/assert`（`GR106`）。
- 验证: `go test ./internal/xenv/service/ -run 'Run|Compose' -v`
- 完成标准: 新增用例全绿，且不依赖真实 SDK 安装；不修改既有测试文件。
- 依赖: T1。

### T3 CLI 命令、注册与装配

- 文件: `internal/cli/run_cmd.go`（新增）、`internal/cli/app.go`（改动：注册命令）、`internal/xenv/xenv.go`（改动：新增 `RunService()`）
- 动作:
  1. `NewRunCmd()`：`Name: "run"`、`Aliases: []string{"exec"}`、`c.AddArg("command", "command and arguments to run", true, true)`（已实测：声明参数后位置参数不再触发 subcommand 报错）。
  2. 选项：可重复的 `-u/--use`、`-p/--path`、`-e/--env` 用 `c.VarOpt` 绑定自定义 `flag.Value` 切片类型（已实测可重复且支持短名与 `--opt=value`）；`-c/--cwd` 用 `c.StrOpt`；`--print` 用 `c.BoolOpt`。
  3. `--use` 值支持逗号分隔（拆分后再进入 `RunOptions.Use`）。
  4. Func 流程：取 `c.Arg("command").Strings()`；缺命令时报用法错误；`--print` 时打印 `RunEnv.Added`、最终 PATH 与命令并返回 0；否则执行并在子进程结束后 `os.Exit(code)`（决策 D7）。
  5. `app.go` 注册 `NewRunCmd()`；`xenv.go` 增加 `RunService()`，复用 `config.Mgr.Init()`、`SDKManager` 与 `SDKService` 的既有构造路径。
- 验证: `go build -o tmp/xenv.exe ./cmd/xenv`；`xenv run --help`、`xenv exec --help` 显示全部选项与参数。
- 完成标准: 帮助信息含 `-u/-p/-e/-c/--print` 与 `command` 参数；`xenv run` 无参数时给出用法错误且不崩溃。
- 依赖: T1（`RunService` 签名）。

### T4 端到端验证（Windows 与 Unix）

- 文件: 无（临时脚本放 `tmp/`，验证后删除）
- 动作:
  1. Windows：`xenv run -u go:<已安装版本> --cwd <临时目录> -- go env GOROOT`；`cmd /c exit 3` 透传；`--print` 不执行；未安装版本、不存在 `--path`、非目录 `--cwd`、缺 `=` 的 `--env` 各报错一次。
  2. 状态不变性：记录 `~/.config/xenv/global.toml`、`session/` 与当前目录 `.xenv.toml` 的内容 hash 与 mtime，执行前后对比一致。
  3. Unix：`GOOS=linux go build` 后用 WSL 运行同一组场景（`sh -c 'exit 3'`、`--cwd`、`--print`）。
  4. 记录每条命令的实际输出作为证据。
- 验证: 上述命令的实际输出；对比结果与设计验收 1-5 一致。
- 完成标准: 验收 1-5 全部可复现；无状态文件被修改；清理 `tmp/` 与 WSL 临时目录。
- 依赖: T3。

### T5 文档

- 文件: `README.md`、`README.zh-CN.md`
- 动作: 新增 `xenv run`/`exec` 小节（选项表、`--` 用法说明、三个示例），并在命令职责表补充该命令；保持与既有文档结构一致。
- 验证: 人工核对示例与实测输出一致（命令、选项名、输出关键行）。
- 完成标准: 双语文档均含命令、选项与示例；无未落地选项被写入文档。
- 依赖: T3、T4（示例须与实测一致）。

## 回滚与恢复

- 每个任务一个本地 atomic commit，提交前核对 `git diff --cached --name-only` 与本任务 owner 文件一致。
- 回滚：`git revert <commit>`；或删除 `run_cmd.go`/`run_service.go`/测试文件并移除 `app.go` 的注册行与 `xenv.go` 的构造函数。无状态残留需要清理（本命令不写任何持久化状态）。
- 恢复点：W1 结束（服务层可单测）、W2 结束（命令可执行）、W3 结束（验收通过）。
- dirty 保护：任一阶段发现 owner 外修改即停止并按 Ownership Conflict 处理。

## 人工 Gate

| Gate | 内容 | 状态 |
|---|---|---|
| G1 计划批准 | 用户批准本计划后进入实施 | 待批准 |
| G2 当前执行请求 | 批准计划不自动授权执行；需单独“开始实施”请求 | 未触发 |
| 外部动作 Gate | 不适用（`host_or_non_offline_action=NOT_APPLICABLE`，无 push/release/deploy/迁移/设备/外部消息） | 不适用 |

## 可追溯性

| 设计条目 | 任务 | 验证 |
|---|---|---|
| 验收 1（`--use/--env/--cwd` 生效） | T1、T3、T4 | T4 Windows/Unix 实测输出 |
| 验收 2（退出码透传、127） | T1、T3、T4 | T4 `cmd /c exit 3`、`sh -c 'exit 3'`、不存在命令 |
| 验收 3（不写状态） | T1（不持有 StateManager）、T4 | T4 状态文件 hash/mtime 对比 |
| 验收 4（参数校验） | T1、T2、T4 | T2 单测 + T4 报错场景 |
| 验收 5（`--print` 不执行） | T1、T3、T4 | T4 `--print` 输出与无副作用确认 |
| 验收 6（单测） | T2 | `go test ./internal/xenv/service/ -run 'Run|Compose'` |
| 验收 7（文档） | T5 | 文档与实测示例核对 |
| D1/D4（命令名与 PATH 顺序） | T1、T3 | T2 单测 + T3 帮助输出 |
| D2（不读写状态、不加载 direnv） | T1、T4 | T4 状态不变性 |
| D3（ENV 覆盖顺序） | T1、T2 | T2 单测 |
| D5（合成 PATH 解析可执行文件） | T1、T4 | T4 用 SDK bin 中的命令验证命中正确版本 |
| D6（`--print` 只打印增量） | T1、T3 | T4 输出检查 |
| D7（退出码落地方式） | T3、T4 | T4 退出码实测 |
| D8（`--` 分隔） | T3、T4 | T3 帮助文案 + T4 无 `--` 时的报错提示 |
| D9（不做信号转发） | T3 | 设计声明，无实现任务 |
| D10（`--cwd` 用 `Cmd.Dir`） | T1、T2、T4 | T2 校验单测 + T4 工作目录实测 |

## 完成 Gate 与剩余工作

完成 Gate（全部满足才可声明完成）：

1. 验收 1-7 均有实测或测试证据。
2. `go build ./...`、`go vet ./...` 通过；新增单测通过；`go test ./...` 的失败集合与实施前基线一致（本机存在 2 个预存失败：`TestEnvSetSaveDirenvFlagWritesXenvToml`、`TestSetupDirenvSkipsSDKsForOtherOS`，已在基线确认与本次改动无关）。
3. 消费本计划声明的 `host_or_non_offline_action=NOT_APPLICABLE`；不产生外部动作。
4. 临时产物（`tmp/` 下的探针与验证脚本、WSL 临时目录）已清理。

剩余工作（不在本次范围）：

- Q1、Q3-Q7 若日后确认需要实现，按 `Semantic Amendment` 返回设计/计划修订与人工 Gate。
- `xenv run` 的并发执行、`--json` 输出、交互式子 shell 等未列入任何波次。
