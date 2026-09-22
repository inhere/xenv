package cli

import (
	"fmt"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv"
)

// PathCmd the xenv path command
var PathCmd = &gcli.Command{
	Name: "path",
	Desc: "Manage PATH environment variable",
	Subs: []*gcli.Command{
		PathAddCmd(),
		PathRemoveCmd(),
		PathListCmd(),
		PathSearchCmd(),
	},
	Aliases: []string{"p"},
	Func: func(c *gcli.Command, args []string) error {
		return listEnvPaths(false)
	},
}

// PathAddCmd command for adding a path to PATH
func PathAddCmd() *gcli.Command {
	return &gcli.Command{
		Name: "add",
		Help: "add [-g] [-S] <path>",
		Desc: "Add a path to PATH environment variable",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&GlobalFlag, "global", "g", false, "Global operation, not the current session")
			c.BoolOpt(&SaveDirenv, "direnv", "s,d", false, "Operate for direnv config .xenv.toml")
			c.BoolOpt(&SystemFlag, "system", "S", false, "Add to the OS user PATH, take effect for new processes")
			c.AddArg("path", "PATH environment value", true)
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

			path := c.Arg("path").String()
			if SystemFlag {
				if err = checkSystemScope(GlobalFlag, SaveDirenv); err != nil {
					return err
				}

				script, err1 := envSvc.AddSystemPath(path)
				if err1 != nil {
					return fmt.Errorf("failed to add path to system PATH: %w", err1)
				}

				fmt.Printf("Added %s to system PATH\n", path)
				printScript(script)
				return nil
			}

			// Add the path
			script, err1 := envSvc.AddPath(path, GetOpFlag())
			if err1 != nil {
				return fmt.Errorf("failed to add path: %w", err1)
			}

			// Save configuration if global
			if GlobalFlag {
				fmt.Printf("Added %s to PATH globally\n", path)
			} else {
				fmt.Printf("Added %s to PATH for current session\n", path)
			}

			printScript(script)
			return nil
		},
	}
}

// PathRemoveCmd command for removing a path from PATH
func PathRemoveCmd() *gcli.Command {
	var pathRmOpts = struct {
		matchMode bool // TODO
	}{}

	return &gcli.Command{
		Name:    "remove",
		Help:    "remove [-g] [-S] <path>",
		Desc:    "Remove a path from PATH environment variable",
		Aliases: []string{"rm", "delete"},
		Config: func(c *gcli.Command) {
			c.BoolOpt(&GlobalFlag, "global", "g", false, "Global operation, not the current session")
			c.BoolOpt(&SaveDirenv, "direnv", "s,d", false, "Operate for direnv config .xenv.toml")
			c.BoolOpt(&pathRmOpts.matchMode, "match", "m", false, "Match mode, remove paths that match the given path")
			c.BoolOpt(&SystemFlag, "system", "S", false, "Remove from the OS user PATH, take effect for new processes")
			c.AddArg("path", "PATH environment value", true)
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

			path := c.Arg("path").String()
			if SystemFlag {
				if err = checkSystemScope(GlobalFlag, SaveDirenv); err != nil {
					return err
				}

				script, err1 := envSvc.RemoveSystemPath(path)
				if err1 != nil {
					return fmt.Errorf("failed to remove path from system PATH: %w", err1)
				}

				fmt.Printf("Removed %s from system PATH\n", path)
				printScript(script)
				return nil
			}

			// Remove the path
			script, err1 := envSvc.RemovePath(path, GetOpFlag())
			if err1 != nil {
				return fmt.Errorf("failed to remove path: %w", err1)
			}

			// Save configuration if global
			if GlobalFlag {
				fmt.Printf("Removed %s from PATH globally\n", path)
			} else {
				fmt.Printf("Removed %s from PATH for current session\n", path)
			}

			printScript(script)
			return nil
		},
	}
}

// PathListCmd command for listing PATH entries
func PathListCmd() *gcli.Command {
	var opts struct {
		System bool
	}
	return &gcli.Command{
		Name:    "list",
		Desc:    "List PATH entries",
		Aliases: []string{"ls"},
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.System, "system", "S", false, "Also list the OS user PATH entries")
		},
		Func: func(c *gcli.Command, args []string) error {
			return listEnvPaths(opts.System)
		},
	}
}

func listEnvPaths(withSystem bool) error {
	// Create env service
	envSvc, err := xenv.EnvService()
	if err != nil {
		return err
	}

	// List PATH entries
	ccolor.Infoln("Global PATH Entries:")
	for i, path := range envSvc.GlobalState().Paths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}
	if len(envSvc.GlobalState().Paths) == 0 {
		fmt.Printf("  - No configuration\n")
	}

	ccolor.Infoln("Session PATH Entries:")
	for i, path := range envSvc.SessionState().Paths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}
	if len(envSvc.SessionState().Paths) == 0 {
		fmt.Printf("  - No configuration\n")
	}

	if !withSystem {
		return nil
	}

	ccolor.Infoln("System PATH Entries:")
	sysPaths, err := envSvc.SystemPaths()
	if err != nil {
		ccolor.Warnf("  failed to read system PATH: %v\n", err)
		return nil
	}
	for i, path := range sysPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}
	if len(sysPaths) == 0 {
		fmt.Printf("  - No configuration\n")
	}
	return nil
}

// PathSearchCmd command for searching PATH entries
func PathSearchCmd() *gcli.Command {
	return &gcli.Command{
		Name:    "search",
		Desc:    "Search for a path in PATH",
		Aliases: []string{"s"},
		Config: func(c *gcli.Command) {
			c.AddArg("value", "value for search in PATH", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			searchTerm := c.Arg("value").String()

			// Create env service
			envSvc, err := xenv.EnvService()
			if err != nil {
				return err
			}

			// Search for the path
			matches := envSvc.SearchPath(searchTerm)
			if len(matches) == 0 {
				fmt.Printf("No paths found containing: %s\n", searchTerm)
			} else {
				fmt.Printf("Paths containing '%s':\n", searchTerm)
				for i, match := range matches {
					fmt.Printf("  %d. %s\n", i+1, match)
				}
			}

			return nil
		},
	}
}
