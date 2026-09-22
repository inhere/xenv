package cli

import (
	"fmt"

	"github.com/gookit/goutil/errorx"
	"github.com/inhere/xenv/internal/xenv"
	"github.com/inhere/xenv/internal/xenv/models"
)

// GetOpFlag 根据参数获取操作标识
func GetOpFlag() models.OpFlag {
	return opFlagFrom(GlobalFlag, SaveDirenv)
}

// checkSystemScope 校验 -S/--system 不能与 -g/--global、-s/-d/--direnv 同时使用
func checkSystemScope(global, saveDirenv bool) error {
	if global || saveDirenv {
		return errorx.New("the -S/--system option can not be used together with -g/--global or -s/--direnv")
	}
	return nil
}

// printScript 输出 --Expression-- 标记后的 shell 执行脚本
func printScript(script string) {
	if script != "" {
		fmt.Printf("%s\n%s\n", xenv.ScriptMark, script)
	}
}

func opFlagFrom(global, saveDirenv bool) models.OpFlag {
	if global {
		return models.OpFlagGlobal
	}
	if saveDirenv {
		return models.OpFlagDirenv
	}
	return models.OpFlagSession
}
