package cli

import (
	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/events"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

// NewApp creates the xenv CLI application.
func NewApp() *gcli.App {
	app := gcli.NewApp(func(app *gcli.App) {
		app.Name = "xenv"
		app.Desc = "Manage local development environments and tools"
	})

	app.On(events.OnAppBindOptsAfter, func(ctx *gcli.HookCtx) bool {
		ctx.App.Flags().BoolVar(&xenvcom.DebugMode, &gcli.CliOpt{
			Name:   "debug",
			Shorts: []string{"d"},
			Desc:   "Enable debug mode. can be XENV_DEBUG_MODE=true",
			DefVal: xenvcom.DebugMode,
		})
		return false
	})

	app.Add(
		ToolsCmd,
		NewUseCmd(),
		NewUnuseCmd(),
		EnvSetCmd(),
		EnvUnsetCmd(),
		EnvCmd,
		PathCmd,
		ConfigCmd,
		ListCmd,
		InitCmd,
		NewShellCmd(),
		ShellHookInitCmd(),
		ShellDirenvCmd(),
	)
	return app
}
