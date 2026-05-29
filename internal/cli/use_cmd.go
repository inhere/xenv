package cli

import (
	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/xenv"
	"github.com/inhere/xenv/internal/xenv/shell"
)

// NewUseCmd the xenv use command
func NewUseCmd() *gcli.Command {
	return &gcli.Command{
		Name: "use",
		Help: "use [-g] <name:version>...",
		Desc: "Switch and activate different versions of SDK",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&GlobalFlag, "global", "g", false, "Global operation, not the current session")
			c.BoolOpt(&SaveDirenv, "save", "s,d", false, "Save change to direnv config .xenv.toml")

			c.AddArg("sdks", "Name of the SDK to activate, allow multi.", true, true)
		},
		Func: func(c *gcli.Command, args []string) error {
			sdkSvc, err := xenv.SDKService()
			if err != nil {
				return err
			}

			useSDKs := c.Arg("sdks").Strings()
			script, err1 := sdkSvc.ActivateSDKs(useSDKs, GetOpFlag())
			if err1 == nil {
				shell.OutputScript(script)
			}
			return err1
		},
	}
}

// NewUnuseCmd the xenv unuse command
func NewUnuseCmd() *gcli.Command {
	return &gcli.Command{
		Name: "unuse",
		Help: "unuse [-g] <name:version>...",
		Desc: "Deactivate specific SDK versions",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&GlobalFlag, "global", "g", false, "Global operation, not the current session")
			c.BoolOpt(&SaveDirenv, "save", "s,d", false, "Save change to direnv config .xenv.toml")
			c.AddArg("sdks", "Name of the SDK to deactivate, allow multi.", true, true)
		},
		Func: func(c *gcli.Command, args []string) error {
			sdkSvc, err := xenv.SDKService()
			if err != nil {
				return err
			}

			unSDKs := c.Arg("sdks").Strings()
			script, err1 := sdkSvc.DeactivateSDKs(unSDKs, GetOpFlag())
			if err1 == nil {
				shell.OutputScript(script)
			}
			return err1
		},
	}
}
