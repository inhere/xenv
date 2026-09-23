package cli

import (
	"fmt"
	"strings"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

var (
	// GlobalFlag option value
	GlobalFlag bool
	SaveDirenv bool
	SystemFlag bool
	DebugMode  bool
)

// EnvCmd the xenv env command
var EnvCmd = &gcli.Command{
	Name: "env",
	Desc: "Manage environment variables",
	Subs: []*gcli.Command{
		EnvSetCmd(),
		EnvUnsetCmd(),
		EnvListCmd(),
	},
	Aliases: []string{"e"},
	Func: func(c *gcli.Command, args []string) error {
		return listEnvs(false)
	},
}

// EnvSetCmd command for setting environment variables
//
// Test run:
//
//	// pwsh
//	$env:XENV_HOOK_SHELL="pwsh"; xenv set TEST003=value003
func EnvSetCmd() *gcli.Command {
	var opts struct {
		Global     bool
		SaveDirenv bool
		System     bool
	}
	return &gcli.Command{
		Name: "set",
		Help: "set [-g] [-s|-d] [-S] KEY=VALUE ...",
		Desc: "Set environment variables",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.SaveDirenv, "direnv", "s,d", false, "Save change to direnv config .xenv.toml")
			c.BoolOpt(&opts.Global, "global", "g", false, "Operate for global config")
			c.BoolOpt(&opts.System, "system", "S", false, "Set to the OS user environment, take effect for new processes")

			c.AddArg("values", "Environment variable pairs, format KEY=VALUE, allow multi. Two args without '=' are read as: name value", true, true)
		},
		Func: func(c *gcli.Command, args []string) error {
			envs := parseSetArgs(c.Arg("values").Strings())

			// Create env service
			envSvc, err := xenv.EnvService()
			if err != nil {
				if outputHookWarningExpression("failed to initialize xenv command", err) {
					return nil
				}
				return err
			}

			if opts.System {
				if err = checkSystemScope(opts.Global, opts.SaveDirenv); err != nil {
					return err
				}

				script, err1 := envSvc.SetSystemEnvs(envs)
				if err1 != nil {
					return fmt.Errorf("failed to set system environment variable: %w", err1)
				}

				for _, item := range envs {
					ccolor.Infof("Set %s to system environment\n", item)
				}
				printScript(script)
				return nil
			}

			// Set the environment variables
			opFlag := opFlagFrom(opts.Global, opts.SaveDirenv)
			script, err := envSvc.SetEnvs(envs, opFlag)
			if err != nil {
				return fmt.Errorf("failed to set environment variable: %w", err)
			}

			// Save configuration if global
			if opFlag == models.OpFlagGlobal {
				for _, item := range envs {
					ccolor.Infof("Set %s globally\n", item)
				}
			} else if opFlag == models.OpFlagDirenv {
				for _, item := range envs {
					ccolor.Infof("Set %s for direnv state\n", item)
				}
			} else {
				for _, item := range envs {
					ccolor.Infof("Set %s for current session\n", item)
				}
			}

			printScript(script)
			return nil
		},
	}
}

// parseSetArgs 解析 set 命令的参数
//
//   - KEY=VALUE ...: 多个环境变量键值对
//   - <name> <value>: 只有两个参数且第一个不含 '=' 时, 按 name value 处理
func parseSetArgs(args []string) []string {
	if len(args) == 2 && !strings.Contains(args[0], "=") {
		return []string{args[0] + "=" + args[1]}
	}
	return args
}

// EnvUnsetCmd command for unsetting environment variables
func EnvUnsetCmd(desc ...string) *gcli.Command {
	var opts struct {
		Global     bool
		SaveDirenv bool
		System     bool
	}
	return &gcli.Command{
		Name: "unset",
		Help: "unset [-g] [-s|-d] [-S] <name...>",
		Desc: "Unset environment variables",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.SaveDirenv, "direnv", "s,d", false, "Operate for direnv config .xenv.toml")
			c.BoolOpt(&opts.Global, "global", "g", false, "Operate for global config")
			c.BoolOpt(&opts.System, "system", "S", false, "Unset from the OS user environment, take effect for new processes")
			c.AddArg("names", "environment key name", true, true)
		},
		Func: func(c *gcli.Command, args []string) error {
			// Create env service
			envSvc, err := xenv.EnvService()
			if err != nil {
				if outputHookWarningExpression("failed to initialize xenv command", err) {
					return nil
				}
				return err
			}

			names := c.Arg("names").Strings()

			if opts.System {
				if err = checkSystemScope(opts.Global, opts.SaveDirenv); err != nil {
					return err
				}

				script, err1 := envSvc.UnsetSystemEnvs(names)
				if err1 != nil {
					return fmt.Errorf("failed to unset system environment variable: %w", err1)
				}

				ccolor.Infof("Unset %s from system environment\n", names)
				printScript(script)
				return nil
			}

			// Unset the environment variables
			opFlag := opFlagFrom(opts.Global, opts.SaveDirenv)
			script, err1 := envSvc.UnsetEnvs(names, opFlag)
			if err1 != nil {
				return fmt.Errorf("failed to set environment variable: %w", err1)
			}

			// Save configuration if global
			if opFlag == models.OpFlagGlobal {
				ccolor.Infof("Unset %s globally\n", names)
			} else if opFlag == models.OpFlagDirenv {
				ccolor.Infof("Unset %s for direnv state\n", names)
			} else {
				ccolor.Infof("Unset %s for current session\n", names)
			}

			printScript(script)
			return nil
		},
	}
}

// EnvListCmd command for listing environment variables
func EnvListCmd() *gcli.Command {
	var opts struct {
		System bool
	}
	return &gcli.Command{
		Name:    "list",
		Desc:    "List environment variables",
		Aliases: []string{"ls"},
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.System, "system", "S", false, "Also list the OS user environment variables")
		},
		Func: func(c *gcli.Command, args []string) error {
			return listEnvs(opts.System)
		},
	}
}

func listEnvs(withSystem bool) error {
	// Create env service
	envSvc, err := xenv.EnvService()
	if err != nil {
		return err
	}

	// List environment variables
	envVars := envSvc.GlobalEnv()
	ccolor.Infoln("Global Environment Variables:")
	for name, envVar := range envVars {
		fmt.Printf("  %s=%s\n", name, envVar)
	}

	if xenvcom.InHookShell() {
		sessVars := envSvc.SessionEnv()
		ccolor.Infoln("Session Environment Variables:")
		for name, envVar := range sessVars {
			fmt.Printf("  %s=%s\n", name, envVar)
		}
	}

	if !withSystem {
		return nil
	}

	ccolor.Infoln("System Environment Variables:")
	sysVars, err := envSvc.SystemEnv()
	if err != nil {
		ccolor.Warnf("  failed to read system environment: %v\n", err)
		return nil
	}
	for name, envVar := range sysVars {
		fmt.Printf("  %s=%s\n", name, envVar)
	}
	return nil
}
