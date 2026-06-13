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

Runtime State 的事实来源是当前 shell 环境，而不是 `global.toml`、`.xenv.toml` 或 session JSON。session JSON 只能表达 Session Context / Session Defaults，不能被当成当前 shell 已经真实生效的 PATH/ENV。

## 第一版范围

1. 只检测 xenv 已知 SDK 的 bin 目录。
2. 不解析 `go version`、`java -version` 等命令输出。
3. 不修改 shell 环境。
4. 不自动修复 PATH。
5. 不把检测结果写回任何 state 文件。

## 检测流程

1. 计算 Effective State。
2. 根据 SDK index 找到 Effective SDK 对应 bin dir。
3. 读取当前进程的 `PATH`。
4. 扫描 `PATH` 中第一个匹配同名 SDK 的 bin dir。
5. 若 runtime bin dir 与 effective bin dir 不一致，输出 warning。

## SDK Bin 匹配

匹配应尽量基于 xenv 已知索引，而不是字符串猜测：

1. 从配置和 SDK local index 得到每个已安装 SDK 的 `BinDirPath()`。
2. 将 `PATH` entry 与这些 bin path 做规范化比较。
3. 同名 SDK 多个版本同时出现在 `PATH` 时，以 `PATH` 中靠前的版本作为 runtime actual。
4. `PATH` 中没有同名 SDK bin path 时，runtime actual 为 missing。

路径比较需要兼容：

- Windows 大小写不敏感路径。
- Windows `\` / `/` 混用。
- Git Bash 暴露的 `/c/...`、`/d/...` 路径与 Windows 路径的等价关系。
- 尾部路径分隔符和重复分隔符。

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

runtime 缺失示例：

```text
[Runtime State]
---------------------------------------------------------------------
go:
  actual:   missing from PATH
  expected: 1.23 from Directory State
  expected bin:
            D:\work\env\devsdk\gosdk\go1.23.12\bin
```

## Warning 规则

输出 warning 的条件：

1. Effective State 中声明了某个 SDK。
2. SDK index 能找到 expected bin dir。
3. Runtime State 中找不到该 SDK，或找到的版本/bin dir 与 expected 不一致。

不输出 warning 的条件：

1. Effective State 没有声明该 SDK。
2. SDK index 里没有 enough information 推导 expected bin dir。
3. Runtime bin dir 与 expected bin dir 规范化后相同。

## 非目标

- 不在 `status --runtime` 中执行 shell hook 或改变当前 shell。
- 不从 session JSON 推断 runtime actual。
- 不通过外部命令输出做版本检测。
- 不解决 PATH 修复；后续可以独立设计 `xenv doctor` 或 `xenv status --fix`。
