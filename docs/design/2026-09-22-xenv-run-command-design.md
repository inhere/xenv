<!-- template_id: design; template_version: 1.1.1 -->
# xenv run 一次性环境执行命令设计

> 状态：Draft 0.2 / 待人工计划批准

## 修订记录

| 版本 | 日期 | 作者 | 摘要 |
|---|---|---|---|
| 0.1 | 2026-09-22 | Pi | 初稿，定义 `xenv run`/`exec` 的命令契约、环境合成规则、执行与退出码语义，并列出待确认事项。 |
| 0.2 | 2026-09-22 | Pi | 用户批准设计并追加 `--cwd` 支持：纳入范围、CLI 契约、执行语义与决策 D10，Q2 关闭；记录批准证据。 |

> 仅语义变化递增版本；纯 identity/provenance/元数据纠正沿用原版本，并在 Git/进度记录中留痕。

## 规划可靠性声明

- `thinking_mode=RIGOROUS`。
- `core_objective`：新增 `xenv run`（别名 `exec`）命令，用命令行选项一次性构造 SDK/PATH/ENV 并执行目标命令，不读不写任何 xenv 状态、不依赖 shell hook。
- `allowed_scope`：`internal/cli`、`internal/xenv/service`、`internal/xenv/models`（仅复用，无 schema 变更）、`internal/xenv/xenv.go`、双语文档与测试。
- `non_goals`：见“范围与非目标”。
- `expansion_policy=DEFER_OR_REQUEST`：范围冻结后新增的 Module、外部协议、生命周期动作只在直接追溯核心验收、安全或可执行唯一性时进入候选，否则记录为 deferred 或请求扩围。
- review budget：Full track，低暴露度（本地开发工具、无生产数据、无出机器动作），评审封顶一轮合并轴评审，不建 review ledger。
- 停止条件：出现会改变用户结果、接口、安全或授权的核心选择时停止并请求确认；无 `CORE_BLOCKING` 时在本设计的人工计划 Gate 停止。

## 背景与目标

`xenv` 目前的激活语义是“状态 + shell hook”：

- `xenv use go:1.24` 会把结果写入 session state（`~/.config/xenv/session/<session_id>.json`），并且只有 shell hook eval `xenv` 输出脚本后当前 shell 才真正生效；不在 hook 环境时只打印提示，不改变任何东西（见 `internal/xenv/service/sdk_service.go` 的 `activateSDKs`）。
- 项目级与全局级分别写入最近的 `.xenv.toml` 与 `~/.config/xenv/global.toml`。

这带来三个用户可见问题：

1. 一次性场景（CI 脚本、临时试用某个版本、在别人机器上跑一次构建）需要先配置状态，用完还要清理。
2. 状态是“粘性”的：只想为一条命令用 `go:1.24`，却会改变当前 shell 会话后续所有命令的行为。
3. 没有 hook 的进程（编辑器任务、CI runner、GUI 启动的进程）无法通过 `xenv use` 获得环境。

目标（用户可见结果）：提供一次性执行入口，只影响被执行的子进程。

```bash
xenv run --use go:1.24,node:22 --path D:/tools/bin --env APP_ENV=local -- go version
xenv exec -u go:1.24 -- go build ./...
xenv run -u go:1.24 --cwd D:/work/proj -- go test ./...
```

- 不改动任何 xenv 状态文件，不依赖 shell hook。
- 命令退出码逐位透传，可直接用于脚本判断。
- Windows、Linux、macOS 行为一致。

## 名词

| 名词 | 含义 |
|---|---|
| 一次性环境 | 只为一次子进程执行构造的环境，进程结束即消失，不落盘 |
| SDK 规格 | `name`、`name:version` 或 `name@version`，缺省版本为 `latest`（`internal/xenv/sdk/version.go`） |
| bin 目录 | SDK 可执行文件目录，`InstalledSDK.BinDirPath()`，由 `ToolChain.install_dir` + `bin_dir` 计算 |
| active_env | SDK 激活时需要额外设置的环境变量，值支持 `{version}`、`{install_dir}` 占位符 |
| 合成环境 | 继承环境 + SDK active_env + `--env` 依次覆盖后的最终环境 |
| 合成 PATH | `--path` 条目 + SDK bin 目录 + 继承 PATH 依次拼接后的最终 PATH |
| xenv 状态 | global state、directory state（`.xenv.toml`）、session state 三类持久化状态 |
| hook | shell 集成，负责 eval xenv 输出脚本使变更作用于当前 shell |

## 范围与非目标

范围：

1. 新增 `xenv run` 命令与 `exec` 别名，选项 `--use/-u`、`--path/-p`、`--env/-e`、`--cwd/-c`、`--print`。
2. SDK 规格解析与本地安装解析（复用现有索引与版本匹配）。
3. 环境与 PATH 合成规则、命令解析、子进程执行、退出码与错误提示。
4. 跨平台行为（Windows 与 Unix）与单元测试、双语文档。

非目标：

1. 不写任何 xenv 状态文件（global/directory/session 均不写），不触发 hook 输出（不打印 `--Expression--`）。
2. 不改动 `use`、`env`、`path`、`shell` 等既有命令语义。
3. 不实现 SDK 下载或安装（`eget` 只读既有索引；未安装版本直接报错）。
4. 不实现交互式子 shell、作业控制、PTY、信号转发协议。
5. 不实现并发执行、批量任务编排、JSON 输出（见“待确认事项”）。

## 已确认事实与规范

代码事实（均已核对当前实现）：

| 事实 | 证据 |
|---|---|
| 版本规格解析支持 `name` / `name:version` / `name@version`，缺省 `latest` | `internal/xenv/sdk/version.go: ParseVersionSpec`、`ParseMultipleVersionSpecs` |
| SDK 解析入口为 `config.FindSDKConfig` + `sdks.MatchSDKByVersion`，`EgetEnable` 时再查合并索引 | `internal/xenv/service/sdk_service.go: checkActivateSDK` |
| 解析结果 `InstalledSDK` 提供 bin 目录与 active_env 渲染 | `internal/xenv/models/sdk_local.go: BinDirPath`、`RenderActiveEnv`；`internal/xenv/models/tool_config.go: ToolChain.FullBinPath`、`RenderActiveEnv` |
| 现有激活只产出 shell 脚本并写状态，不产出进程环境 | `internal/xenv/service/sdk_service.go: activateSDKs`（`GenRemThenAddPaths` / `GenSetEnvs` + `state.UseSDKsWithParams`） |
| PATH 分隔符与拆分/拼接有平台适配 | `internal/xenv/xenvcom/hook.go: PathSep`、`internal/util/fsutil.go: SplitPath`、`JoinPaths` |
| 路径条目支持 `windows:` / `linux:` / `darwin:` 前缀过滤 | `internal/xenv/models/path_spec.go: FilterPathsForGOOS` |
| 命令注册集中在 `internal/cli/app.go` | `internal/cli/app.go: NewApp` |
| gcli 会把位置参数之后的选项重排解析；`--` 终止解析，其后 token 原样保留 | gcli v3.8.0 `gflag/reorder.go: rearrangeArgs`、`gflag/flags.go: parseOne`、`gcli.Command.RawArgs()` |
| 应用退出码来自 `errorx.ErrorCoder`，普通 error 返回 0；错误经 `EvtAppRunError` 打印 `ERROR:` 前缀 | `internal/cli/app.go`、gcli `app.go: Run` |

规则引用：`SR1204`（>3 文件或 >100 行业务改动需实施前范围确认）、`SR1403`/`SR1405`（最小实现、无证据不抽象）、`SR1407`（最小可执行验证）、`SR1206`（提交前缀）、`SR1211`（本地原子提交）、`GR104`（文件规模）、`GR106`（测试断言用 `goutil/x/assert`）。

偏离：无。

## 总体方案

### 命令契约

```text
xenv run [options] -- <command> [args...]
xenv exec [options] -- <command> [args...]
```

| 选项 | 说明 | 取值规则 |
|---|---|---|
| `-u, --use <spec,...>` | 一次性激活的 SDK 规格 | 可重复，也可逗号分隔；支持 `go`、`go:1.24`、`go@1.24` |
| `-p, --path <dir>` | 追加到 PATH 最前面的目录 | 可重复；目录必须存在 |
| `-e, --env <KEY=VALUE>` | 设置/覆盖环境变量 | 可重复；键名统一大写，`KEY=` 表示空值 |
| `-c, --cwd <dir>` | 子进程的工作目录 | 目录必须存在且为目录；不改变 xenv 自身进程的工作目录 |
| `--print` | 只打印本次合成结果与将执行的命令，不执行 | 只打印增量部分，不打印继承环境 |
| `--` | 结束 xenv 选项，其后为命令与参数 | 命令自身带 `-`/`--` 参数时必须使用 |

`xenv run` 至少需要一个命令；缺少命令时报用法错误。

### 环境合成规则

ENV 优先级（后者覆盖前者）：

```text
继承环境(os.Environ)  ->  SDK active_env（按 --use 顺序）  ->  --env（按输入顺序）
```

PATH 组成（前置即优先）：

```text
--path 条目（按输入顺序） ; SDK bin 目录（按 --use 顺序） ; 继承 PATH
```

- 条目按平台分隔符拼接（Windows `;`，Unix `:`）。
- 去重：同一目录只保留首次出现的条目，比较时统一分隔符与末尾分隔符、忽略大小写（Windows 语义，复用 `internal/xenv/sysenv/pathlist.go` 的 `normalizeWinPath` 思路，Unix 侧按原样比较）。
- 空条目丢弃。
- 继承 PATH 中的条目原样保留，不展开 `%VAR%`，不改写用户已有值。

### 执行语义

1. 解析并校验选项；`--use` 逐个解析为 `InstalledSDK`，任一步失败即整体失败，不执行命令。
2. 合成环境与 PATH。
3. 用合成后的 PATH 解析命令可执行文件（见决策 D5）。
4. 以继承的 stdin/stdout/stderr 启动子进程，并显式传入合成环境；`--cwd` 通过子进程的工作目录生效（见决策 D10）。
5. 透传退出码：子进程退出码原样返回；命令找不到返回 127；xenv 自身参数/解析错误返回非 0 且不执行命令。

### 与既有命令的关系

| 对比项 | `xenv use` | `xenv run` |
|---|---|---|
| 作用范围 | 当前 shell 会话（hook 后） | 仅本次子进程 |
| 持久化 | 写 session/direnv/global 状态 | 不写任何文件 |
| 依赖 hook | 是 | 否 |
| 退出码 | xenv 自身退出码 | 子进程退出码 |

## 架构

```text
internal/cli/run_cmd.go         命令定义、选项绑定、参数切分、退出码落地
        |
        v
internal/xenv/service/run_service.go
        |-- RunService.ResolveSDKs  -> 复用 SDKService.checkActivateSDK（同包私有方法）
        |-- RunService.BuildEnv    -> 纯函数合成（ENV/PATH）
        |-- RunService.Run         -> 解析可执行文件 + exec
        |
        v
internal/xenv/models（ToolChain / InstalledSDK，只读复用）
internal/xenv/sdk（ParseVersionSpec，只读复用）
```

新增与改动文件（预计）：

| 文件 | 类型 | 职责 |
|---|---|---|
| `internal/cli/run_cmd.go` | 新增 | 命令与选项定义、`--` 切分、结果输出、退出码处理 |
| `internal/xenv/service/run_service.go` | 新增 | SDK 解析、环境合成、命令解析与执行 |
| `internal/xenv/service/run_service_test.go` | 新增 | 合成规则、参数校验、规格解析错误的单测 |
| `internal/xenv/xenv.go` | 改动 | 增加 `RunService()` 构造函数，与 `EnvService()`/`SDKService()` 并列 |
| `internal/cli/app.go` | 改动 | 注册 `run` 命令 |
| `README.md` / `README.zh-CN.md` | 改动 | 命令与示例文档 |

设计约束：

- 合成逻辑（ENV 覆盖、PATH 拼接与去重）实现为不依赖 OS 的纯函数，便于在任意平台单测。
- 不引入新的第三方依赖；仅使用标准库与现有 `goutil`/`gcli`。
- `RunService` 不持有 `StateManager`，从构造上保证不会写状态文件。

## 关键流程

正常路径：

```text
xenv run -u go:1.24,node:22 -p D:/tools/bin -e APP_ENV=local --cwd D:/work/proj -- go version

1. cli 解析选项，取 -- 之后的 ["go","version"]
2. service 解析 go:1.24 / node:22 -> InstalledSDK（bin 目录 + active_env）
3. 合成 ENV：os.Environ + GOROOT/... + APP_ENV=local
4. 合成 PATH：D:/tools/bin ; <go bin> ; <node bin> ; 原 PATH
5. 用合成 PATH 解析 "go" -> <go bin>/go.exe
6. 启动子进程（工作目录 D:/work/proj，继承 stdio，传入合成环境），等待结束
7. 以子进程退出码结束 xenv 进程
```

异常路径：

| 场景 | 行为 | 退出码 |
|---|---|---|
| `--use` 引用的 SDK 未在 config 中定义 | 报错并提示 `sdk <name> config is not defined` | 非 0，不执行 |
| 版本未安装 | 报错 `sdk <name>:<version> is not installed locally`，提示可用版本 | 非 0，不执行 |
| `--path` 目录不存在 | 报错 `path does not exist: <dir>` | 非 0，不执行 |
| `--cwd` 目录不存在或不是目录 | 报错 `cwd is not a directory: <dir>` | 非 0，不执行 |
| `--env` 缺少 `=` 或键名非法 | 报错并给出正确格式 | 非 0，不执行 |
| 命令不在合成 PATH 中 | 报错 `command not found: <name>` | 127 |
| 子进程非零退出 | 不额外打印 ERROR，直接透传 | 子进程退出码 |
| 命令参数未用 `--` 分隔且含 `-` 开头参数 | gcli 会当作 xenv 选项解析并报未定义选项；错误信息提示改用 `--` | 非 0，不执行 |

与 hook 的交互：`xenv run` 在任何环境（含 hook shell）都只影响子进程，不输出 `--Expression--`，因此不会被 hook 误当作需要 eval 的脚本。

## 安全、数据、运维与回滚

安全：

- 不写任何持久化状态（不写状态文件、注册表、shell 启动文件），失败时不留残留。
- `--print` 只打印由选项引入的增量（SDK bin/active_env、`--path`、`--env`）与最终 PATH，不打印继承环境，避免把 CI 中的密钥变量写入日志。
- `--cwd` 只设置子进程工作目录，不调用 `os.Chdir`，因此不改变 xenv 自身对相对路径（`--path`、`--env` 值中的相对路径）的解析基准，也不会影响同进程内其它逻辑。
- 子进程环境由显式合成，不继承 `XENV_HOOK_SHELL`、`XENV_SESSION_ID` 之外的 xenv 内部变量做任何写操作。

数据：无 schema、无迁移、无缓存文件。

运维：无网络访问；无后台进程；无新增部署步骤。

回滚：删除 `run_cmd.go` 与 `app.go` 的注册行即恢复原状，无状态残留需要清理。

## 决策

| 编号 | 决策 | 理由与被否方案 |
|---|---|---|
| D1 | 命令名 `run`，别名 `exec` | 与 `mise run`/`npx` 习惯一致；`run` 与现有命令不冲突（现有命令见 `internal/cli/app.go`）。不采用 `exec` 作主名以免与 shell 内建 `exec` 语义混淆 |
| D2 | 不读不写任何 xenv 状态，不加载 `.xenv.toml` | 这是本命令的核心语义：一次性、无副作用。需要项目状态的场景仍用 hook 流程 |
| D3 | ENV 覆盖顺序：继承 → SDK active_env → `--env` | 命令行显式选项优先级最高；SDK 提供的 `GOROOT` 等不应被继承环境中的旧值压制 |
| D4 | PATH 顺序：`--path` → SDK bin → 继承 PATH | 与现有 `path add`、SDK 激活的“前置即优先”语义一致 |
| D5 | 用合成 PATH 解析可执行文件：临时设置进程 `PATH` 后调用 `exec.LookPath`，随后立即恢复 | 复用标准库的平台解析规则（Windows `PATHEXT`、Unix 可执行位），避免自实现 30+ 行的查找逻辑。CLI 进程单线程且即将退出，副作用可控。备选：自实现 `lookPathIn(dirs, name)`，代价是复制平台规则 |
| D6 | `--print` 只打印增量，不打印完整环境 | 避免泄漏继承的敏感变量；需要完整环境时可用 `--print` 结合 shell 自身命令 |
| D7 | 退出码在命令内用 `os.Exit` 落地，不经由 gcli 错误通道 | gcli 的错误通道会打印 `ERROR:` 前缀且普通 error 返回 0，不适合透传子进程退出码；`main` 本身也是 `os.Exit(app.Run(...))`，语义等价 |
| D8 | 命令参数必须用 `--` 分隔（文档与错误提示中说明） | gcli 的 `rearrangeArgs` 会把 `-x` 之类 token 重排为 xenv 选项；`--` 之后原样保留是框架既有语义。不采用关闭重排的全局配置，避免影响其他命令 |
| D9 | 不实现信号转发 | Unix 下父子同进程组，Ctrl+C 会同时到达；Windows 控制台同理。实现转发会显著增加平台分支，收益不足（`SR1403`） |
| D10 | `--cwd` 通过子进程工作目录（`exec.Cmd.Dir`）实现，不调用 `os.Chdir` | 保持 xenv 自身解析基准不变，避免全局副作用；代价是 `--path` 等选项中的相对路径仍相对调用目录解析，需在文档中写明 |

## 待确认事项

| 编号 | 问题 | 备选与影响 |
|---|---|---|
| Q1 | 是否支持 `--unset KEY`（一次性删除变量） | 影响 CLI 契约与合成顺序；不实现时用户可用 `--env KEY=` 置空 |
| Q3 | `--path` 是否允许不存在的目录 | 当前设计按 `path add` 一致性要求目录存在；放宽则更灵活但掩盖拼写错误 |
| Q4 | 是否允许叠加当前 `.xenv.toml` / session 状态（如 `--with-direnv`） | 与 D2 冲突，需要明确开关语义；默认不叠加 |
| Q5 | 是否需要 `--json` 输出合成结果供脚本消费 | 影响输出契约；不实现时可解析 `--print` 文本 |
| Q6 | 别名集合是否包含 `x` | 影响命令注册与文档 |
| Q7 | `--use` 是否支持版本范围/别名（如 `lts`、`latest`）以外的匹配策略开关 | 现有 `allow_up_match` 配置已在匹配层生效，是否需要 per-command 覆盖待定 |

Q2 已在 0.2 确认（支持 `--cwd`，语义见决策 D10），编号保持不变以便追溯；Q1、Q3-Q7 仍未确认，未确认项按“不实现”处理。

## 结论与人工计划 Gate

本设计定义了一个无副作用的一次性执行入口：`xenv run`/`exec` 通过 `--use`/`--path`/`--env`/`--cwd` 合成环境并执行目标命令，不写状态、不依赖 hook、透传退出码。

批准证据：用户于 2026-09-22 在会话中批准本设计，并追加 `--cwd` 支持要求（对应 0.2 修订）。批准只授权进入计划；实施需要另行批准实施计划。

Delivery Track 判定为 Full（新增外部 CLI 接口与选项契约），治理暴露度低（本地开发工具、无生产数据、无出机器动作），评审封顶一轮合并轴评审。实施前需按 `SR1204` 完成范围确认（预计 6 个文件、约 320 行业务代码），并取得实施计划批准。

下一步：编写实施计划（`docs/plans/2026-09-22-xenv-run-command-implementation.md`），在人工计划 Gate 再次停止。

批准只授权进入计划阶段；实施需要实施计划的人工批准，评审结论不构成批准。
