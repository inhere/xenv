package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
		configPath := config.GetDefaultConfigPath()

		// Ensure config directory exists
		configDir := filepath.Dir(configPath)
		if err := util.EnsureDir(configDir); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Try to load existing config, if it exists
		if _, err := os.Stat(configPath); err == nil {
			if err := cfgMgr.LoadConfig(configPath); err != nil {
				fmt.Printf("Warning: failed to load existing config: %v\n", err)
			}
		} else {
			// If no existing config, save the default config
			if err := cfgMgr.SaveConfig(configPath); err != nil {
				return fmt.Errorf("failed to save default config: %w", err)
			}
			if err := cfgMgr.LoadConfig(configPath); err != nil {
				return fmt.Errorf("failed to load created config: %w", err)
			}
			fmt.Printf("Created default configuration at: %s\n", configPath)
		}

		// Ensure required directories exist
		if err := util.EnsureDir(cfgMgr.Config.BinDir); err != nil {
			return fmt.Errorf("failed to create bin directory: %w", err)
		}

		if err := util.EnsureDir(cfgMgr.Config.ShellHooksDir); err != nil {
			return fmt.Errorf("failed to create shell scripts directory: %w", err)
		}

		fmt.Println("Xenv initialization completed successfully!")
		fmt.Printf("Configuration file: %s\n", configPath)
		fmt.Printf("Bin directory: %s\n", cfgMgr.Config.BinDir)
		fmt.Printf("Shell scripts directory: %s\n", cfgMgr.Config.ShellHooksDir)

		return nil
	},
}
