package xenvcmd

import (
	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/xenvcmd/subcmd"
	"github.com/inhere/xenv/pkg/xenv/xenvcom"
)

// XEnvCmd the main xenv command
var XEnvCmd = &gcli.Command{
	Name: "xenv",
	// Aliases: []string{"xenv"},
	Desc: "Manage local development environments and tools, similar to mise and vfox",
	Help: `
Quick commands:
  <info>set</>    Quick exec the 'env set' subcommand
  <info>unset</>  Quick exec the 'env unset' subcommand
`,
	Subs: []*gcli.Command{
		subcmd.ToolsCmd,
		subcmd.NewUseCmd(),
		subcmd.NewUnuseCmd(),
		subcmd.EnvSetCmd(),
		subcmd.EnvUnsetCmd(),
		subcmd.EnvCmd,
		subcmd.PathCmd,
		subcmd.ConfigCmd,
		subcmd.ListCmd,
		subcmd.InitCmd,
		subcmd.NewShellCmd(),
		subcmd.ShellHookInitCmd(),
		subcmd.ShellDirenvCmd(),
	},
	// Configure the command
	Config: func(c *gcli.Command) {
		// Add global options for xenv command if needed
		c.BoolOpt(&subcmd.GlobalFlag, "global", "g", false, "Operate for global config")
		c.BoolOpt(&xenvcom.DebugMode, "debug", "d", xenvcom.DebugMode, "Enable debug mode. can be XENV_DEBUG_MODE=true")
	},
}
