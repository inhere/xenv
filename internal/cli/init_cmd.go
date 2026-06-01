package cli

import (
	"fmt"

	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/util"
	"github.com/inhere/xenv/internal/xenv/config"
)

// InitCmd the xenv config init command
var InitCmd = &gcli.Command{
	Name: "init",
	Desc: "Initialize xenv configuration and environment",
	Func: func(c *gcli.Command, args []string) error {
		cfgMgr := config.Mgr
		created, err := cfgMgr.EnsureConfig()
		if err != nil {
			return fmt.Errorf("failed to initialize configuration: %w", err)
		}
		if created {
			fmt.Printf("Created default configuration at: %s\n", cfgMgr.Config.ConfigFile())
		} else {
			fmt.Printf("Configuration already exists at: %s\n", cfgMgr.Config.ConfigFile())
		}

		// Ensure required directories exist
		if err := util.EnsureDir(cfgMgr.Config.BinDir); err != nil {
			return fmt.Errorf("failed to create bin directory: %w", err)
		}

		if err := util.EnsureDir(cfgMgr.Config.ShellHooksDir); err != nil {
			return fmt.Errorf("failed to create shell scripts directory: %w", err)
		}

		fmt.Println("Xenv initialization completed successfully!")
		fmt.Printf("Configuration file: %s\n", cfgMgr.Config.ConfigFile())
		fmt.Printf("Bin directory: %s\n", cfgMgr.Config.BinDir)
		fmt.Printf("Shell scripts directory: %s\n", cfgMgr.Config.ShellHooksDir)

		return nil
	},
}
