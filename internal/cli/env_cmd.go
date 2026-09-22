package cli

import (
	"fmt"

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
//	$env:XENV_HOOK_SHELL="pwsh"; xenv set TEST003 value003
func EnvSetCmd() *gcli.Command {
	var opts struct {
		Global     bool
		SaveDirenv bool
		System     bool
	}
	return &gcli.Command{
		Name: "set",
		Help: "set [-g] [-s|-d] [-S] <name> <value>",
		Desc: "Set an environment variable",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.SaveDirenv, "direnv", "s,d", false, "Save change to direnv config .xenv.toml")
			c.BoolOpt(&opts.Global, "global", "g", false, "Operate for global config")
			c.BoolOpt(&opts.System, "system", "S", false, "Set to the OS user environment, take effect for new processes")

			c.AddArg("name", "environment key name", true)
			c.AddArg("value", "environment value", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			name := c.Arg("name").String()
			value := c.Arg("value").String()

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

				script, err1 := envSvc.SetSystemEnv(name, value)
				if err1 != nil {
					return fmt.Errorf("failed to set system environment variable: %w", err1)
				}

				ccolor.Infof("Set %s=%s to system environment\n", name, value)
				printScript(script)
				return nil
			}

			// Set the environment variable
			opFlag := opFlagFrom(opts.Global, opts.SaveDirenv)
			script, err := envSvc.SetEnv(name, value, opFlag)
			if err != nil {
				return fmt.Errorf("failed to set environment variable: %w", err)
			}

			// Save configuration if global
			if opFlag == models.OpFlagGlobal {
				ccolor.Infof("Set %s=%s globally\n", name, value)
			} else if opFlag == models.OpFlagDirenv {
				ccolor.Infof("Set %s=%s for direnv state\n", name, value)
			} else {
				ccolor.Infof("Set %s=%s for current session\n", name, value)
			}

			printScript(script)
			return nil
		},
	}
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
