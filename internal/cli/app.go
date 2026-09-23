package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/gookit/gcli/v3"
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

	app.On(gcli.EvtAppBindOptsAfter, func(ctx *gcli.HookCtx) bool {
		ctx.App.Flags().BoolVar(&xenvcom.DebugMode, &gcli.CliOpt{
			Name:   "debug",
			Shorts: []string{"d"},
			Desc:   "Enable debug mode. can be XENV_DEBUG_MODE=true",
			DefVal: xenvcom.DebugMode,
		})
		return false
	})
	app.On(gcli.EvtAppRunError, func(ctx *gcli.HookCtx) bool {
		if errV := ctx.Get("err"); errV != nil {
			if err, ok := errV.(error); ok {
				_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			}
		}
		return true
	})
	// 未知子命令直接以用法错误码退出, 避免脚本无法察觉命令拼错
	app.On(gcli.EvtCmdSubNotFound, func(ctx *gcli.HookCtx) bool {
		name, _ := ctx.Get("name").(string)
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s - subcommand %q is not found\n", ctx.Cmd.Name, name)
		os.Exit(2)
		return true
	})

	app.Add(
		SDKCmd,
		CheckCmd,
		NewUseCmd(),
		NewUnuseCmd(),
		EnvCmd,
		PathCmd,
		NewRunCmd(),
		ConfigCmd,
		StatusCmd(),
		NewShellCmd(),
		ShellHookInitCmd(),
		ShellDirenvCmd(),
	)

	envCategory := "Quick for Environment"
	envSetCmd := EnvSetCmd()
	envSetCmd.Desc = "Set environment variables. equals to call `env set`"
	envSetCmd.Category = envCategory
	envUnsetCmd := EnvUnsetCmd()
	envUnsetCmd.Desc = "Unset environment variables. equals to call `env unset`"
	envUnsetCmd.Category = envCategory

	app.Add(envSetCmd, envUnsetCmd)
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
