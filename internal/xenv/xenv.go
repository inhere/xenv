package xenv

import (
	"fmt"

	"github.com/inhere/xenv/internal/xenv/config"
	"github.com/inhere/xenv/internal/xenv/manager"
	"github.com/inhere/xenv/internal/xenv/service"
)

// ScriptMark 输出的脚本必须添加标记，前面部分为message, 后面部分为脚本
const ScriptMark = "--Expression--"

// Init initializes the xenv config and state
func Init() error {
	// Initialize configuration
	if err := config.Mgr.Init(); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	if err := InitState(); err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	if err := SDKMgr().Init(config.Config()); err != nil {
		return fmt.Errorf("failed to initialize sdk manager: %w", err)
	}
	return nil
}

var stateMgr = manager.NewStateManager()

// State returns the state manager
func State() *manager.StateManager {
	return stateMgr
}

// InitState initializes the state manager
func InitState() error {
	return stateMgr.Init()
}

var sdkMgr *manager.SDKManager

func SDKMgr() *manager.SDKManager {
	if sdkMgr == nil {
		sdkMgr = manager.NewSDKManager(config.DefaultPaths().SDKLocalIndexFile)
	}
	return sdkMgr
}

func SDKService() (*service.SDKService, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	return service.NewSDKService(config.Config(), stateMgr, SDKMgr()), nil
}

func EnvService() (*service.EnvService, error) {
	// Initialize configuration
	if err := config.Mgr.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize configuration: %w", err)
	}

	if err := InitState(); err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Create env manager
	return service.NewEnvService(config.Mgr.Config, stateMgr), nil
}
