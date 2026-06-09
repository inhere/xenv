package cli

import (
	"fmt"

	"github.com/gookit/cliui/show/title"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

// ListCmd the xenv list command
var ListCmd = &gcli.Command{
	Name:    "list",
	Desc:    "List local SDKs, ENV variables, or PATH entries",
	Aliases: []string{"ls"},
	Subs: []*gcli.Command{
		ListSDKCmd(),
		ListEnvCmd(),
		ListPathCmd(),
		ListStateCmd(),
		ListAllCmd(),
	},
	Func: func(c *gcli.Command, _ []string) error {
		return handleListActivity(false)
	},
}

// ListSDKCmd lists SDKs.
func ListSDKCmd() *gcli.Command {
	return &gcli.Command{
		Name:    "sdk",
		Desc:    "List local installed SDKs",
		Aliases: []string{"sdks"},
		Func: func(c *gcli.Command, args []string) error {
			return listSDKs()
		},
	}
}

// ListEnvCmd lists environment variables
func ListEnvCmd() *gcli.Command {
	return &gcli.Command{
		Name: "env",
		Desc: "List xenv environment variables",
		Func: func(c *gcli.Command, args []string) error {
			return listEnvs()
		},
	}
}

// ListPathCmd lists PATH entries
func ListPathCmd() *gcli.Command {
	return &gcli.Command{
		Name: "path",
		Desc: "List PATH entries by xenv set",
		Func: func(c *gcli.Command, args []string) error {
			return listEnvPaths()
		},
	}
}

// ListStateCmd lists active SDKs and settings
func ListStateCmd() *gcli.Command {
	var listActOpts = struct {
		Group bool `flag:"shorts=t;desc=List activity states and group by global, dir, session"`
	}{}

	return &gcli.Command{
		Name:    "state",
		Desc:    "List active SDKs, ENV, and PATH",
		Aliases: []string{"st", "status"},
		Config: func(c *gcli.Command) {
			c.MustFromStruct(&listActOpts)
		},
		Func: func(c *gcli.Command, _ []string) error {
			return handleListActivity(listActOpts.Group)
		},
	}
}

func handleListActivity(groupInfo bool) error {
	// Load activity state
	if err := xenv.InitState(); err != nil {
		return fmt.Errorf("failed to load activity state: %w", err)
	}

	tl := title.New("", func(t *title.Title) {
		t.Color = "ylw1"
		t.PercentWidth = 80
		t.PaddingLR = false
		t.ShowBorder = true
	})
	if !groupInfo {
		tl.ShowNew("Activity States")
		listActivity(xenv.State().Merged())
		return nil
	}

	tl.ShowNew("[Global State]")
	global := xenv.State().Global()
	if global.IsEmpty() {
		fmt.Println("No global state found")
	} else {
		listActivity(global)
	}

	dirStates := xenv.State().DirStates()
	if len(dirStates) > 0 {
		fmt.Println()
		tl.ShowNew("[Directory States]")
		for _, dirState := range dirStates {
			fmt.Println(" - form:", dirState.File)
			listActivity(dirState)
		}
	}

	if xenvcom.InHookShell() {
		sess := xenv.State().Session()
		fmt.Println()
		tl.ShowNew("[Session State]")
		fmt.Println(" - from:", sess.File)
		if sess.IsEmpty() {
			fmt.Println("No session state found")
		} else {
			listActivity(sess)
		}
	}
	return nil
}

func listActivity(state *models.ActivityState) {
	ccolor.Cyanln("Active Develop SDKs:")
	for name, version := range state.SDKs {
		ccolor.Printf("  <green>%10s</> => %s\n", name, version)
	}

	ccolor.Cyanln("\nActive Env Variables:")
	for name, value := range state.Envs {
		ccolor.Printf("  <green>%s</>=%s\n", name, value)
	}

	ccolor.Cyanln("\nActive PATH Entries:")
	for i, path := range state.Paths {
		ccolor.Printf("  <green>%d</>. %s\n", i+1, path)
	}

	ccolor.Cyanln("\nTool Requirements:")
	for name, requirement := range state.ToolRequirements {
		ccolor.Printf("  <green>%s</> => %s\n", name, requirement)
	}
}

// ListAllCmd lists everything
func ListAllCmd() *gcli.Command {
	return &gcli.Command{
		Name: "all",
		Desc: "List all SDKs, env vars, and paths",
		Func: func(c *gcli.Command, args []string) error {
			// This would call all the other list commands
			fmt.Println("This would list all items - implementation needed")
			return nil
		},
	}
}

func listSDKs() error {
	sdkSvc, err := xenv.SDKService()
	if err != nil {
		return err
	}

	return sdkSvc.ListSDKs(false)
}
