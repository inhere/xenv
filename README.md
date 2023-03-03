# Kite

`kite` - Personal developer tool command application.

![app cmds](docs/images/kite-in-wsl.png)

## Git 仓库

* [https://github.com/inhere/kite](https://github.com/inhere/kite)  PHP 版本，功能较为完善，已开发使用较久。
* [https://github.com/inhere/kite-go](https://github.com/inhere/kite-go) Go 语言版本，暂时只有常用功能。

## 主要功能

* git 常用命令操作
* gitlab 常用命令操作
* github 常用命令操作
* 字符串处理工具: 分析，格式化，提取信息，转换
* json 处理工具: 格式化，查找，过滤等
* go, php, java 代码生成，转换等
* json, yaml, sql 格式化，转换
* 系统、环境信息查看
* 快速运行内置脚本
* 快速运行系统命令
* 文件查找匹配，处理
* 批量运行命令
* 文档搜索、查看等

## 安装

### Quick install

```bash
curl https://raw.githubusercontent.com/inhere/kite-go/main/cmd/install.sh | bash
```

**From proxy**

```shell
curl https://ghproxy.com/https://raw.githubusercontent.com/inhere/kite-go/main/cmd/install.sh | bash
```

### Install by go

```bash
go install github.com/inhere/kite-go/cmd/kite
```

## Build

```bash
make install
# or
go build -o $GOPAHT/bin/kite ./cmd/kite
```

## Develop

### Dev build

```shell
KITE_INIT_LOG=debug go run ./cmd/kite
```

### Install to GOBIN

```bash
make kit2gobin
# or
make kite2gobin
```

## Gookit Packages

- https://github.com/gookit/config
- https://github.com/gookit/rux
- https://github.com/gookit/gcli
- https://github.com/gookit/ini

## Refers

- https://github.com/bitfield/script
