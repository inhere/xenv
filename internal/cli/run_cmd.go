package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/xenv"
	"github.com/inhere/xenv/internal/xenv/service"
)

// stringList 可重复选项的值列表
type stringList []string

// String 返回列表的可读形式
func (s *stringList) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

// Set 追加一个选项值, 支持重复指定
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// NewRunCmd the xenv run command
//
// 一次性构造 SDK/ENV/PATH 环境并执行命令, 不读写任何 xenv 状态, 不依赖 shell hook
func NewRunCmd() *gcli.Command {
	var opts struct {
		Use   stringList
		Paths stringList
		Envs  stringList
		Cwd   string
		Print bool
	}

	return &gcli.Command{
		Name:    "run",
		Desc:    "Run a command with a one-shot SDK/ENV/PATH environment",
		Aliases: []string{"exec"},
		Help: `
<cyan>One-shot environment:</>
  xenv run -u go:1.24,node:22 -p ./bin -e APP_ENV=local -- go build ./...
  xenv exec -u go:1.24 --cwd D:/work/proj -- go test ./...

<cyan>Notes:</>
  - only the child process is affected, no xenv state file is written
  - options of the target command must be placed after "--"
  - --print shows the resolved env and command without running it
`,
		Config: func(c *gcli.Command) {
			c.VarOpt(&opts.Use, "use", "u", "SDK specs to activate, repeatable, eg: go:1.24,node:22")
			c.VarOpt(&opts.Paths, "path", "p", "Directory to prepend to PATH, repeatable")
			c.VarOpt(&opts.Envs, "env", "e", "Environment variable KEY=VALUE, repeatable")
			c.StrOpt(&opts.Cwd, "cwd", "c", "", "Working directory for the command")
			c.BoolOpt(&opts.Print, "print", "", false, "Print the resolved env and command, do not run")
			c.AddArg("command", "command and arguments to run, place after --", false, true)
		},
		Func: func(c *gcli.Command, args []string) error {
			command := c.Arg("command").Strings()
			if len(command) == 0 {
				exitWithError(errors.New("no command to run, usage: xenv run [options] -- <command> [args...]"))
			}

			runSvc, err := xenv.RunService()
			if err != nil {
				exitWithError(err)
			}

			runEnv, err := runSvc.BuildEnv(service.RunOptions{
				Use:   splitOptionList(opts.Use),
				Paths: opts.Paths,
				Envs:  opts.Envs,
				Cwd:   opts.Cwd,
			})
			if err != nil {
				exitWithError(err)
			}

			if opts.Print {
				printRunEnv(runEnv, command)
				return nil
			}

			code, err := runSvc.Run(runEnv, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			}

			// 直接以子进程退出码结束, 避免 gcli 错误通道添加额外输出
			os.Exit(code)
			return nil
		},
	}
}

// exitWithError 输出错误信息并以用法错误码结束进程
//
// NOTE: run 需要可靠的非零退出码供脚本判断, 因此不沿用其它命令 exit 0 的错误行为
func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(2)
}

// splitOptionList 拆分逗号分隔的选项值, 忽略空项
func splitOptionList(values []string) []string {
	var list []string
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			if item = strings.TrimSpace(item); item != "" {
				list = append(list, item)
			}
		}
	}
	return list
}

// printRunEnv 输出合成后的环境与将执行的命令
func printRunEnv(runEnv *service.RunEnv, command []string) {
	for _, sdkItem := range runEnv.SDKs {
		fmt.Printf("SDK %s\n", sdkItem.ID)
	}
	for _, item := range runEnv.Envs {
		fmt.Printf("ENV %s\n", item)
	}

	fmt.Printf("PATH %s\n", runEnv.Path)
	if runEnv.Cwd != "" {
		fmt.Printf("CWD %s\n", runEnv.Cwd)
	}
	fmt.Printf("COMMAND %s\n", strings.Join(command, " "))
}
