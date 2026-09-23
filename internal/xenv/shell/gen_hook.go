package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/maputil"
	"github.com/inhere/xenv/internal/util"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

// XenvScriptGenerator xenv Shell脚本生成器实现
type XenvScriptGenerator struct {
	// cfg *models.Configuration
	shell ShType
}

// NewScriptGenerator creates a new ShellGenerator
func NewScriptGenerator(shellType ShType) *XenvScriptGenerator {
	return &XenvScriptGenerator{shell: shellType}
}

// endregion
// region Generate Init Scripts
//

// GenHookScripts 生成 Shell Hook 初始化脚本代码
func (sg *XenvScriptGenerator) GenHookScripts(ps *models.GenInitScriptParams) (string, error) {
	switch sg.shell {
	case Bash:
		return sg.generateBashScripts(ps), nil
	case Zsh:
		return sg.generateZshScripts(ps), nil
	case Pwsh:
		return sg.generatePwshScripts(ps), nil
	default:
		return sg.generateCmdScripts(ps), nil
	}
}

func (sg *XenvScriptGenerator) GenSourceProjectScript(projectDir string) string {
	projectDir = filepath.ToSlash(projectDir)
	switch sg.shell {
	case Bash, Zsh:
		return fmt.Sprintf("source \"%s/.xenv.sh\"\n", projectDir)
	case Pwsh:
		return fmt.Sprintf(". \"%s/.xenv.ps1\"\n", projectDir)
	default:
		return ""
	}
}

// InstallToProfile 安装 Shell Hook 脚本到配置文件(eg: .bashrc, .zshrc)
//
// 返回实际写入的配置文件路径
func (sg *XenvScriptGenerator) InstallToProfile(pwshProfile string) (string, error) {
	profilePath, err := sg.profilePath(pwshProfile)
	if err != nil {
		return "", err
	}

	content, err := readProfileFile(profilePath)
	if err != nil {
		return "", err
	}

	updated := upsertHookBlock(content, sg.hookEvalLine())
	if err = writeProfileFile(profilePath, updated); err != nil {
		return "", err
	}
	return profilePath, nil
}

// profilePath 返回需要写入的 shell 配置文件路径
func (sg *XenvScriptGenerator) profilePath(pwshProfile string) (string, error) {
	switch sg.shell {
	case Bash, Zsh:
		return util.NormalizePath(sg.shell.ProfilePath()), nil
	case Pwsh:
		if pwshProfile == "" {
			return "", fmt.Errorf("please provide the pwsh profile path, eg: --profile $PROFILE.CurrentUserAllHosts")
		}
		return util.NormalizePath(pwshProfile), nil
	}
	return "", fmt.Errorf("install to profile is not supported for shell %s, please add the hook manually", sg.shell)
}

// hookEvalLine 返回在配置文件中加载 hook 的命令
func (sg *XenvScriptGenerator) hookEvalLine() string {
	if sg.shell == Pwsh {
		return fmt.Sprintf("Invoke-Expression (& %s shell --type pwsh)", xenvcom.BinCommand)
	}
	return fmt.Sprintf(`eval "$(%s shell --type %s)"`, xenvcom.BinCommand, sg.shell)
}

// hookBlockStart/hookBlockEnd hook 托管块的开始/结束标记
const (
	hookBlockStart = "# >>> xenv hook >>>"
	hookBlockEnd   = "# <<< xenv hook <<<"
)

// upsertHookBlock 写入或替换配置文件中由 xenv 管理的 hook 块
func upsertHookBlock(content, evalLine string) string {
	block := hookBlockStart + "\n" + evalLine + "\n" + hookBlockEnd

	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	start, end := -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case start < 0 && trimmed == hookBlockStart:
			start = i
		case start >= 0 && trimmed == hookBlockEnd:
			end = i
		}
	}

	// 已存在托管块: 替换块内容
	if start >= 0 && end > start {
		out := append([]string{}, lines[:start]...)
		out = append(out, strings.Split(block, "\n")...)
		out = append(out, lines[end+1:]...)
		return strings.Join(out, "\n") + "\n"
	}

	// 新文件或空文件
	if len(lines) == 1 && lines[0] == "" {
		return block + "\n"
	}
	return strings.Join(lines, "\n") + "\n\n" + block + "\n"
}

// readProfileFile 读取配置文件内容, 文件不存在时返回空内容
func readProfileFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read %s: %w", filePath, err)
	}
	return string(data), nil
}

// writeProfileFile 写入配置文件, 保留原有文件权限
func writeProfileFile(filePath, content string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(filePath); err == nil {
		mode = info.Mode().Perm()
	}

	if err := fsutil.MkParentDir(filePath); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, []byte(content), mode); err != nil {
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}
	return nil
}

// endregion
// region Generate Snippets
//

// GenSetEnvs 批量生成环境变量设置脚本代码
func (sg *XenvScriptGenerator) GenSetEnvs(envs map[string]string) string {
	var ss []string
	for name, value := range envs {
		ss = append(ss, sg.GenSetEnv(name, value))
	}
	return strings.Join(ss, "\n")
}

// GenUnsetEnvs 批量生成环境变量删除脚本代码
func (sg *XenvScriptGenerator) GenUnsetEnvs(names []string) string {
	var ss []string
	for _, name := range names {
		ss = append(ss, sg.GenUnsetEnv(name))
	}
	return strings.Join(ss, "\n")
}

// GenSetEnv 生成环境变量设置脚本代码
func (sg *XenvScriptGenerator) GenSetEnv(name, value string) string {
	name = strings.ToUpper(name)
	switch sg.shell {
	case Bash, Zsh:
		return fmt.Sprintf("export %s=%s\n", name, shQuote(value))
	case Pwsh:
		return fmt.Sprintf("$Env:%s=%s;\n", name, pwshQuote(value))
	default:
		return fmt.Sprintf("os.setenv('%s', %s)\n\n", name, luaQuote(value))
	}
}

// GenUnsetEnv 删除环境变量的脚本代码
func (sg *XenvScriptGenerator) GenUnsetEnv(name string) string {
	name = strings.ToUpper(name)
	switch sg.shell {
	case Bash, Zsh:
		return fmt.Sprintf("unset %s\n", name)
	case Pwsh:
		return fmt.Sprintf("Remove-Item Env:%s -ErrorAction SilentlyContinue\n", name)
	default:
		return fmt.Sprintf("os.unsetenv('%s')\n", name)
	}
}

// GenAddPath 添加 PATH 脚本代码（添加到 PATH 的第一个位置）
func (sg *XenvScriptGenerator) GenAddPath(path string) string {
	fmtPath := util.FormatShellPathFor(path, string(sg.shell))
	switch sg.shell {
	case Bash, Zsh:
		return fmt.Sprintf("export PATH=%s:$PATH\n", shQuote(fmtPath))
	case Pwsh:
		return fmt.Sprintf("$Env:PATH=%s + $Env:PATH\n", pwshQuote(fmtPath+";"))
	default:
		return fmt.Sprintf("os.setenv('PATH', %s)\n", luaQuote(fmtPath+";%PATH%"))
	}
}

// GenAddPaths 一次添加多个到 PATH 的脚本代码
func (sg *XenvScriptGenerator) GenAddPaths(paths []string) string {
	newPath := util.JoinPaths(paths)
	switch sg.shell {
	case Bash, Zsh:
		return fmt.Sprintf("export PATH=%s:$PATH\n", shQuote(newPath))
	case Pwsh:
		return fmt.Sprintf("$Env:PATH=%s + $Env:PATH\n", pwshQuote(newPath+";"))
	default:
		return fmt.Sprintf("os.setenv('PATH', %s)\n", luaQuote(newPath+";%PATH%"))
	}
}

// GenSetPath 设置 PATH 脚本代码
func (sg *XenvScriptGenerator) GenSetPath(paths []string) string {
	newPath := util.JoinPaths(paths)
	switch sg.shell {
	case Bash, Zsh:
		return fmt.Sprintf("export PATH=%s\n", shQuote(newPath))
	case Pwsh:
		return fmt.Sprintf("$Env:PATH=%s;\n", pwshQuote(newPath))
	default:
		return fmt.Sprintf("os.setenv('PATH', %s)\n\n", luaQuote(newPath))
	}
}

// GenRemovePaths 生成批量删除 PATH 的脚本代码
func (sg *XenvScriptGenerator) GenRemovePaths(paths []string) (script string, notFounds []string) {
	var newPaths []string
	osPathList := util.SplitPath(os.Getenv("PATH"))

	_, newPaths, notFounds = DiffRemovePaths(osPathList, paths)
	if len(newPaths) > 0 {
		script = sg.GenSetPath(newPaths)
	}
	return
}

// GenRemThenAddPaths 生成批量删除后再新增的 PATH 的脚本代码
func (sg *XenvScriptGenerator) GenRemThenAddPaths(rmPaths, addPaths []string) (script string) {
	var newPaths []string
	osPathList := util.SplitPath(os.Getenv("PATH"))

	_, newPaths, _ = DiffRemovePaths(osPathList, rmPaths)
	if len(newPaths) > 0 {
		if len(addPaths) > 0 {
			newPaths = append(addPaths, newPaths...)
		}
		script = sg.GenSetPath(newPaths)
	} else if len(addPaths) > 0 {
		script = sg.GenAddPaths(addPaths)
	}
	return
}

// endregion
// region Helper methods
//

func (sg *XenvScriptGenerator) addCommonForLinuxShell(sb *strings.Builder, ps *models.GenInitScriptParams) {
	// 添加全局环境变量
	if len(ps.Envs) > 0 {
		sb.WriteString("  # Add global ENV variables from xenv\n")
		maputil.EachTypedMap(ps.Envs, func(key, value string) {
			sb.WriteString(fmt.Sprintf("  export %s=%s\n", strings.ToUpper(key), shQuote(value)))
		})
	}

	// 添加全局PATH条目
	if len(ps.Paths) > 0 {
		sb.WriteString("  # Add global PATH from xenv\n")
		var fmtPaths []string
		for _, path := range ps.Paths {
			fmtPaths = append(fmtPaths, shQuote(util.FormatShellPathFor(path, string(sg.shell))))
		}
		sb.WriteString(fmt.Sprintf("  export PATH=%s:$PATH\n", strings.Join(fmtPaths, ":")))
	}

	// 添加全局别名
	if len(ps.ShellAliases) > 0 {
		sb.WriteString("  # Add global aliases from xenv\n")
		maputil.EachTypedMap(ps.ShellAliases, func(key, value string) {
			sb.WriteString(fmt.Sprintf("  alias %s=%s\n", key, shQuote(value)))
		})
	}

}

func shQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// pwshQuote 生成 PowerShell 单引号字符串, 内部单引号使用 ” 转义
func pwshQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// luaQuote 生成 clink(lua) 单引号字符串, 转义反斜杠与单引号
func luaQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return "'" + strings.ReplaceAll(value, "'", `\'`) + "'"
}

func shQuotePathExpr(path string) string {
	if path == "~" {
		return `"${HOME}"`
	}
	if strings.HasPrefix(path, "~/") {
		return `"${HOME}/` + strings.ReplaceAll(path[2:], `"`, `\"`) + `"`
	}
	return shQuote(path)
}
