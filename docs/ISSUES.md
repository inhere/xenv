# ISSUES


## [x] 260604.1 项目下的 .xenv.toml 被更新问题

已修复：`init-direnv`/`SetupDirenv` 自动读取已有 `.xenv.toml` 时只生成激活脚本，不再把解析后的真实 SDK 版本回写到项目配置；显式 `xenv use -s` 仍会保存目录配置。

go 项目，go.mod 片段:

```go
go 1.24

toolchain go1.24.6

```

项目下的 .xenv.toml：

```toml
paths = [
]

[sdks]
go = "1.24" # 问题：进入目录后被改成了 1.24.6

[envs]
[tools]
```
