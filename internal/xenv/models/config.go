package models

// Configuration 代表用户的配置信息，包含工具管理设置、路径配置、环境激活状态等数据
// TODO 将外部工具独立出去， xenv 只管理 ENV, PATH, SDK
type Configuration struct {
	// ID              string                  `json:"id"`
	// tools 可执行文件链接目录 默认: ~/.local/bin
	BinDir string `json:"bin_dir"`
	// 是否启用 eget SDK store 作为附加来源
	EgetEnable bool `json:"eget_enable"`
	// eget SDK store 文件路径
	EgetStoreFile string `json:"eget_store_file"`
	// direnv 激活时是否检查 [tools] 要求
	CheckToolsOnDirenv bool `json:"check_tools_on_direnv"`
	// direnv 激活时是否 source 项目脚本
	SourceProjectScripts bool `json:"source_project_scripts"`
	// 快速配置 shell 命令别名, 会自动注入到shell环境
	ShellAliases map[string]string `json:"shell_aliases"`
	// shell hooks 脚本目录。 默认: ~/.config/xenv/hooks/
	ShellHooksDir string `json:"shell_hooks_dir"`
	// 全局环境变量 - 首次初始化生效，后续通过命令设置即可
	GlobalEnv map[string]string `json:"global_env"`
	// 全局PATH条目 - 首次初始化生效，后续通过命令设置即可
	GlobalPaths []string `json:"global_paths"`
	// 设置了完整版本号，是否允许向上匹配版本 eg: 1.23.1
	//
	// default: 1 see xenvcom.UpMatchOne
	//
	//  0: 不允许，严格完全匹配 eg: 1.23.1
	//  1: 向上一位匹配  eg: 1.23.1, 1.23.2, 1.23.3, ...
	//  2: 向上两位匹配  eg: 1.24.x, 1.25.x, 1.26.x, ...
	//  9: 允许所有高版本(只要比需要的版本高就可以) eg: 1.23.1, 1.24.x
	AllowUpMatch uint8 `json:"allow_up_match"`
	// 可管理的工具链列表
	//  - sdks 和 tools 差异是：sdk 允许本地同时存在多个版本，tools 只允许一个版本
	SDKs []ToolChain `json:"sdks"`
	// internal fields
	configFile string
	configDir  string
}

// IsDefinedSDK returns true if the SDK configuration is defined
func (c *Configuration) IsDefinedSDK(name string) bool {
	// Check if the tool is installed
	toolFound := false
	for _, tool := range c.SDKs {
		if tool.Name == name {
			toolFound = true
			break
		}
	}
	return toolFound
}

// FindSDKConfig returns the SDK configuration if it is defined
func (c *Configuration) FindSDKConfig(name string) *ToolChain {
	for _, tool := range c.SDKs {
		if tool.Name == name {
			return &tool
		}
	}
	return nil
}

func (c *Configuration) SDKNames() []string {
	names := make([]string, len(c.SDKs))
	for i, tool := range c.SDKs {
		names[i] = tool.Name
	}
	return names
}

func (c *Configuration) ConfigFile() string {
	return c.configFile
}

// SetConfigFile sets the config.yaml configuration file path
func (c *Configuration) SetConfigFile(filePath string) {
	c.configFile = filePath
}

// ConfigDir returns the directory path where the configuration file is located
func (c *Configuration) ConfigDir() string {
	return c.configDir
}

// SetConfigDir sets the directory path where the configuration file is located
func (c *Configuration) SetConfigDir(dirPath string) {
	c.configDir = dirPath
}
