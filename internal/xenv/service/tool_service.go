package service

import (
	"fmt"

	"github.com/gookit/goutil/errorx"
	"github.com/inhere/xenv/internal/xenv/manager"
	"github.com/inhere/xenv/internal/xenv/models"
)

// ToolService is a compatibility wrapper kept until CLI Task 4 removes tools commands.
type ToolService struct {
	*SDKService
}

func NewToolService(config *models.Configuration, state *manager.StateManager, sdkMgr *manager.SDKManager) *ToolService {
	return &ToolService{SDKService: NewSDKService(config, state, sdkMgr)}
}

func (ts *ToolService) Register(name string, version string, url string, bin string) error {
	return errorx.Raw("tool register is no longer supported; use local SDK indexing instead")
}

func (ts *ToolService) ListAll(showAll bool) error {
	return ts.ListSDKs(showAll)
}

func (ts *ToolService) UpdateTool(name, version string) error {
	return errorx.Raw("tool update is no longer supported; manage SDK installations outside xenv")
}

func (ts *ToolService) GetTool(name string) *models.ToolChain {
	return ts.config.FindSDKConfig(name)
}

func (ts *ToolService) InstallTool(name, version string) error {
	return errorx.Raw("tool install is no longer supported; manage SDK installations outside xenv")
}

func (ts *ToolService) Uninstall(name, version string) error {
	return fmt.Errorf("tool uninstall is no longer supported; remove SDK %s:%s outside xenv", name, version)
}
