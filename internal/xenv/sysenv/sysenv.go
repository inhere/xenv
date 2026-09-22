// Package sysenv 管理操作系统的用户级环境变量和 PATH。
//
// 与 xenv 自身的 session/direnv/global 状态不同, 这里的改动会持久化到操作系统,
// 保存后新启动的进程可以直接读取, 不再依赖 xenv shell hook:
//
//   - windows: 用户环境注册表 HKCU\Environment
//   - linux/macOS: 当前 shell 的用户启动文件(bash: ~/.bashrc, zsh: ~/.zshrc)中的 xenv 托管块
//
// 各平台实现提供相同的函数:
//
//   - SetVar、UnsetVar: 设置和删除环境变量
//   - AddPath、RemovePath: 添加和删除 PATH 条目
//   - EnvVars、PathList: 读取当前保存的值
package sysenv
