package config

import (
	"fmt"
	"os"
	"path/filepath"

	goyaml "github.com/goccy/go-yaml"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/toml"
	"github.com/gookit/config/v2/yaml"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

const (
	// DefaultBinDir is the default directory for installed tools
	DefaultBinDir = "~/.local/bin"
)

// Manager handles loading and saving configuration
type Manager struct {
	cfgInit bool
	Config  *models.Configuration
}

// Mgr is the global ConfigManager instance
var Mgr = NewConfigManager()

func Config() *models.Configuration {
	return Mgr.Config
}

// NewConfigManager creates a new ConfigManager with default configuration
func NewConfigManager() *Manager {
	paths := DefaultPaths()
	return &Manager{
		Config: &models.Configuration{
			BinDir:               DefaultBinDir,
			EgetEnable:           false,
			EgetStoreFile:        "",
			CheckToolsOnDirenv:   false,
			SourceProjectScripts: false,
			ShellAliases:         make(map[string]string),
			ShellHooksDir:        paths.ShellHooksDir,
			// env
			GlobalEnv:   make(map[string]string),
			GlobalPaths: []string{},
			// sdk
			SDKs: []models.ToolChain{},
			// other
			AllowUpMatch: xenvcom.UpMatchOne,
		},
	}
}

// Init initializes load the configuration data
func (cm *Manager) Init() error {
	if cm.cfgInit {
		return nil
	}
	cm.cfgInit = true
	_, err := cm.EnsureConfig()
	return err
}

func (cm *Manager) EnsureConfig() (created bool, err error) {
	configPath := GetDefaultConfigPath()
	if _, statErr := os.Stat(configPath); statErr == nil {
		return false, cm.LoadConfig(configPath)
	} else if !os.IsNotExist(statErr) {
		return false, statErr
	}

	if err := cm.SaveConfig(configPath); err != nil {
		return false, err
	}
	if err := cm.LoadConfig(configPath); err != nil {
		return true, err
	}
	return true, nil
}

// LoadConfig loads configuration from the specified file
func (cm *Manager) LoadConfig(configPath string) error {
	cfg := config.New("xenv", config.WithTagName("json"), config.ParseEnv)
	cfg.AddDriver(yaml.Driver)
	cfg.AddDriver(toml.Driver)

	// Load the configuration file
	err := cfg.LoadFiles(configPath)
	if err != nil {
		return err
	}

	// Load other configuration values like tools, global environment, etc.
	err = cfg.Decode(&cm.Config)
	cm.Config.SetConfigFile(configPath)
	cm.Config.SetConfigDir(filepath.Dir(configPath))
	return err
}

// SaveConfig saves the configuration to the specified file
func (cm *Manager) SaveConfig(configPath string) error {
	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Marshal the configuration to YAML
	configData, err := goyaml.MarshalWithOptions(cm.Config, goyaml.Indent(2))
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	// Write to the file
	return os.WriteFile(configPath, configData, 0644)
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	return DefaultPaths().ConfigFile
}

func GetDefaultConfigDir() string {
	return DefaultPaths().ConfigDir
}
