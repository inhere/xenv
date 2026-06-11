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
		return listEnvs()
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
	}
	return &gcli.Command{
		Name: "set",
		Help: "set [-g] [-s|-d] <name> <value>",
		Desc: "Set an environment variable",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.SaveDirenv, "direnv", "s,d", false, "Save change to direnv config .xenv.toml")
			c.BoolOpt(&opts.Global, "global", "g", false, "Operate for global config")

			c.AddArg("name", "environment key name", true)
			c.AddArg("value", "environment value", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			name := c.Arg("name").String()
			value := c.Arg("value").String()

			// Create env service
			envSvc, err := xenv.EnvService()
			if err != nil {
				return err
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

			if script != "" {
				fmt.Printf("%s\n%s\n", xenv.ScriptMark, script)
			}
			return nil
		},
	}
}

// EnvUnsetCmd command for unsetting environment variables
func EnvUnsetCmd(desc ...string) *gcli.Command {
	var opts struct {
		Global     bool
		SaveDirenv bool
	}
	return &gcli.Command{
		Name: "unset",
		Help: "unset [-g] [-s|-d] <name...>",
		Desc: "Unset environment variables",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.SaveDirenv, "direnv", "s,d", false, "Operate for direnv config .xenv.toml")
			c.BoolOpt(&opts.Global, "global", "g", false, "Operate for global config")
			c.AddArg("names", "environment key name", true, true)
		},
		Func: func(c *gcli.Command, args []string) error {
			// Create env service
			envSvc, err := xenv.EnvService()
			if err != nil {
				return err
			}

			names := c.Arg("names").Strings()

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

			if script != "" {
				fmt.Printf("%s\n%s\n", xenv.ScriptMark, script)
			}
			return nil
		},
	}
}

// EnvListCmd command for listing environment variables
func EnvListCmd() *gcli.Command {
	return &gcli.Command{
		Name:    "list",
		Desc:    "List environment variables",
		Aliases: []string{"ls"},
		Func: func(c *gcli.Command, args []string) error {
			return listEnvs()
		},
	}
}

func listEnvs() error {
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
	return nil
}
