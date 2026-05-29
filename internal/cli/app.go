package cli

import (
	"fmt"
	"time"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/events"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

var (
	version   string
	gitHash   string
	buildTime string
)

// SetBuildInfo sets the build information for the application.
func SetBuildInfo(versionStr, gitHashStr, buildTimeStr string) {
	version = versionStr
	gitHash = gitHashStr
	buildTime = normalizeBuildTime(buildTimeStr)
}

// NewApp creates the xenv CLI application.
func NewApp() *gcli.App {
	app := gcli.NewApp(func(app *gcli.App) {
		app.Name = "xenv"
		app.Desc = "Manage local development environments and SDK activation"
	})
	app.Version = buildVersionString()

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
		SDKCmd,
		CheckCmd,
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

const compactTimeLayout = "2006-01-02T15:04:05"

func normalizeBuildTime(value string) string {
	for _, layout := range []string{
		time.RFC3339,
		"2006/01/02-15:04:05",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format(compactTimeLayout)
		}
	}
	return value
}

func buildVersionString() string {
	if gitHash == "" && buildTime == "" {
		return version
	}
	return fmt.Sprintf("%s (%s, %s)", version, gitHash, buildTime)
}
