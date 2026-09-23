package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gookit/cliui/show"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/envutil"
	"github.com/inhere/xenv/internal/xenv/config"
)

var configOpts = struct {
	edit bool
}{}

// ConfigCmd the xenv config command
var ConfigCmd = &gcli.Command{
	Name:    "config",
	Desc:    "Manage xenv configuration",
	Aliases: []string{"cfg"},
	Subs: []*gcli.Command{
		InitCmd,
		ConfigSetCmd(),
		ConfigGetCmd(),
		ConfigExportCmd(),
		ConfigImportCmd(),
	},
	Config: func(c *gcli.Command) {
		// builtin editors: TODO
		//  - 通用: vim, helix, nvim
		//  - Linux: nano, vi
		//  - Windows: notepad
		c.BoolOpt(&configOpts.edit, "edit", "e", false, "Edit the configuration file in the default editor")
	},
	Func: func(c *gcli.Command, args []string) error {
		// Initialize load config file
		if err := config.Mgr.Init(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		cfg := config.Mgr.Config

		// 打开配置文件编辑器
		if configOpts.edit {
			return openConfigInEditor(configPathOf(config.Mgr))
		}

		c.Infoln("Loading config file:", cfg.ConfigFile())

		// Display current configuration
		managedSdkKey := fmt.Sprintf("4. Managed SDKs(%d)", len(cfg.SDKs))
		show.AList("【Current Xenv Configuration】:", map[string]any{
			"1. Default Bin Dir": cfg.BinDir,
			"2. Shell Hooks Dir": cfg.ShellHooksDir,
			managedSdkKey:        cfg.SDKNames(),
		})

		return nil
	},
}

// ConfigSetCmd command for setting configuration values
func ConfigSetCmd() *gcli.Command {
	// var configSetOpts = struct {
	// 	configPath string
	// }{}

	return &gcli.Command{
		Name: "set",
		Desc: "Set a configuration value",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "configuration key name", true)
			c.AddArg("value", "configuration value", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			name := c.Arg("name").String()
			value := c.Arg("value").String()

			// Initialize load config file
			if err := config.Mgr.Init(); err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			cfgMgr := config.Mgr

			// Set the configuration value based on the name
			switch name {
			case "bin_dir":
				cfgMgr.Config.BinDir = value
			case "shell_hooks_dir":
				cfgMgr.Config.ShellHooksDir = value
			default:
				return fmt.Errorf("unknown configuration option: %s", name)
			}

			// Save the configuration
			if err := cfgMgr.SaveConfig(configPathOf(cfgMgr)); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Set %s=%s\n", name, value)
			return nil
		},
	}
}

// ConfigGetCmd command for getting configuration values
func ConfigGetCmd() *gcli.Command {
	return &gcli.Command{
		Name: "get",
		Desc: "Get a configuration value",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "configuration key name", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			// Initialize load config file
			if err := config.Mgr.Init(); err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			cfgMgr := config.Mgr
			name := c.Arg("name").String()

			value, ok := cfgMgr.GetValue(name)
			if !ok {
				return fmt.Errorf("config key %q not found", name)
			}

			return printConfigGetValue(name, value)
		},
	}
}

func printConfigGetValue(name string, value any) error {
	switch val := value.(type) {
	case map[string]any, map[any]any, []any:
		data, err := json.MarshalIndent(val, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format config value %q: %w", name, err)
		}
		fmt.Printf("%s=%s\n", name, data)
	default:
		fmt.Printf("%s=%v\n", name, val)
	}
	return nil
}

// ConfigExportCmd command for exporting configuration
func ConfigExportCmd() *gcli.Command {
	var writeTo string
	return &gcli.Command{
		Name: "export",
		Desc: "Export configuration",
		Config: func(c *gcli.Command) {
			c.StrOpt(&writeTo, "to", "w", "stdout", `export configuration to the target.
stdout     -  default output for json format.
file       -  export to xenv_config_export.{format}, default for zip.
file path  -  custom the export output target.
			`)
			c.AddArg("format", "export format, allow: zip, json").WithDefault("json")
		},
		Func: func(c *gcli.Command, args []string) error {
			format := c.Arg("format").String()
			// Validate format
			if format != "zip" && format != "json" {
				return fmt.Errorf("unsupported export format: %s (use 'zip' or 'json')", format)
			}

			// Initialize load config file
			if err := config.Mgr.Init(); err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Create exporter
			exporter := config.NewExporter(config.Mgr)
			exportPath := writeTo
			switch strings.ToLower(writeTo) {
			case "stdout":
				exportPath = "STDOUT"
				if format == "zip" {
					exportPath = "xenv_config_export.zip"
				}
			case "file":
				exportPath = "xenv_config_export." + format
			}

			// Export the configuration
			if err := exporter.Export(exportPath, format); err != nil {
				return fmt.Errorf("failed to export configuration: %w", err)
			}

			fmt.Printf("Configuration exported to: %s\n", exportPath)
			return nil
		},
	}
}

// ConfigImportCmd command for importing configuration
func ConfigImportCmd() *gcli.Command {
	return &gcli.Command{
		Name: "import",
		Desc: "Import configuration from file",
		Config: func(c *gcli.Command) {
			c.AddArg("path", "path to import configuration from", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			importPath := c.Arg("path").String()

			// Initialize load config file
			if err := config.Mgr.Init(); err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			cfgMgr := config.Mgr

			// Create importer
			importer := config.NewImporter(cfgMgr)

			// Import the configuration
			if err := importer.Import(importPath); err != nil {
				return fmt.Errorf("failed to import configuration: %w", err)
			}

			// Save the imported configuration
			if err := cfgMgr.SaveConfig(configPathOf(cfgMgr)); err != nil {
				return fmt.Errorf("failed to save imported configuration: %w", err)
			}

			fmt.Printf("Configuration imported from: %s\n", importPath)
			return nil
		},
	}
}

// configPathOf 返回当前加载的配置文件路径, 未加载时回退到默认路径
func configPathOf(cfgMgr *config.Manager) string {
	if path := cfgMgr.Config.ConfigFile(); path != "" {
		return path
	}
	return config.GetDefaultConfigPath()
}

// openConfigInEditor 使用系统编辑器打开配置文件
func openConfigInEditor(filePath string) error {
	editor := envutil.Getenv("XENV_EDITOR", envutil.Getenv("VISUAL", os.Getenv("EDITOR")))
	if editor == "" {
		editor = defaultEditor()
	}

	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], filePath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor %s: %w", parts[0], err)
	}
	return nil
}

// defaultEditor 返回当前平台的默认编辑器
func defaultEditor() string {
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
