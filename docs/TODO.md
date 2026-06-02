# TODO

<!--
简单的直接使用一行 checklist 说明即可。
需要附带较长说明的，使用标题+说明方式新建。使用emoji 表情状态图标(wait: ⏳|ing: 🔄|done: ✅)
-->

- [x] 现在 xenv 配置shell init 后，进入目录 就会生成 .xenv.toml 文件。如何默认不生成？
  - 或者默认写入到 ~/.config/xenv.projects.json 文件中, 只有手动执行 xenv init/use 才会生成 .xenv.toml 文件。
- [x] xenv 的临时会话文件按 时间生成的，会导致 临时会话文件数量增加。改为按进入的目录路径生成。
- [ ] xenv env set 支持设置环境变量到系统环境, xenv path add 支持添加路径到系统环境
  - 方便使用，避免每次手动去编辑设置环境变量和路径。
- [ ] xenv allow_up_match 向上匹配 value=2, 9 还未实现
- [x] xenv config get 功能完善一下，支持类似 eget config get 可以使用任意key path查询
