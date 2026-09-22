package sysenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// xenv 托管块的开始/结束标记. 该块由 xenv 维护, 不要手动添加其它内容
const (
	rcBlockStart = "# >>> xenv system env >>>"
	rcBlockEnd   = "# <<< xenv system env <<<"
)

// rcFileFor 根据 shell 名称解析需要写入的用户启动文件
//
// 目前只支持 Unix 上的 bash/zsh 等 POSIX shell
func rcFileFor(shellName, homeDir string) (string, error) {
	name := ""
	if trimmed := strings.TrimSpace(shellName); trimmed != "" {
		name = strings.ToLower(filepath.Base(strings.ReplaceAll(trimmed, `\`, "/")))
		name = strings.TrimSuffix(name, ".exe")
	}

	switch name {
	case "zsh":
		return filepath.Join(homeDir, ".zshrc"), nil
	case "bash":
		return filepath.Join(homeDir, ".bashrc"), nil
	case "", "sh", "dash", "ash", "ksh":
		return filepath.Join(homeDir, ".profile"), nil
	}
	return "", fmt.Errorf("system environment is not supported for shell %q, only bash and zsh are supported", shellName)
}

// upsertBlockLine 在 xenv 托管块中新增或更新一行, 托管块不存在时自动创建
func upsertBlockLine(filePath string, match func(string) bool, line string) error {
	lines, err := readRcLines(filePath)
	if err != nil {
		return err
	}

	start, end := rcBlockRange(lines)
	if start < 0 {
		lines = appendRcBlock(lines, line)
	} else {
		lines = upsertRcBlockLine(lines, start, end, match, line)
	}
	return writeRcLines(filePath, lines)
}

// removeBlockLine 删除 xenv 托管块中匹配的行, 块内没有条目时移除整个托管块
//
// return 是否找到并删除了匹配行
func removeBlockLine(filePath string, match func(string) bool) (found bool, err error) {
	lines, err := readRcLines(filePath)
	if err != nil {
		return false, err
	}

	start, end := rcBlockRange(lines)
	if start < 0 {
		return false, nil
	}

	var kept []string
	for _, line := range lines[start+1 : end] {
		if match(line) {
			found = true
			continue
		}
		kept = append(kept, line)
	}
	if !found {
		return false, nil
	}

	if len(kept) > 0 {
		out := append([]string{}, lines[:start+1]...)
		out = append(out, kept...)
		out = append(out, lines[end:]...)
		return true, writeRcLines(filePath, out)
	}

	// 托管块已经没有条目, 连同标记和前面的空行一起移除
	head := lines[:start]
	if len(head) > 0 && head[len(head)-1] == "" {
		head = head[:len(head)-1]
	}
	out := append([]string{}, head...)
	out = append(out, lines[end+1:]...)
	return true, writeRcLines(filePath, out)
}

// blockContent 返回 xenv 托管块内的行, 块不存在时返回 nil
func blockContent(filePath string) ([]string, error) {
	lines, err := readRcLines(filePath)
	if err != nil {
		return nil, err
	}

	start, end := rcBlockRange(lines)
	if start < 0 {
		return nil, nil
	}
	return lines[start+1 : end], nil
}

// blockEnvVars 解析托管块中的环境变量, PATH 条目行由 blockPaths 解析
func blockEnvVars(lines []string) map[string]string {
	envs := make(map[string]string)
	for _, line := range lines {
		if name, value, ok := rcEnvEntry(line); ok {
			envs[name] = value
		}
	}
	return envs
}

// blockPaths 解析托管块中的 PATH 条目
func blockPaths(lines []string) []string {
	var paths []string
	for _, line := range lines {
		if path, ok := rcPathValue(line); ok {
			paths = append(paths, path)
		}
	}
	return paths
}

// rcBlockRange 返回 xenv 托管块的标记行位置
//
// note: 缺少结束标记的块视为无效, 返回 (-1, -1)
func rcBlockRange(lines []string) (start, end int) {
	start = -1
	for i, line := range lines {
		switch {
		case start < 0 && line == rcBlockStart:
			start = i
		case start >= 0 && line == rcBlockEnd:
			return start, i
		}
	}
	return -1, -1
}

// appendRcBlock 在文件末尾追加新的托管块
func appendRcBlock(lines []string, line string) []string {
	var block []string
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		block = append(block, "")
	}
	block = append(block, rcBlockStart, line, rcBlockEnd)
	return append(lines, block...)
}

// upsertRcBlockLine 在托管块内更新或追加一行
func upsertRcBlockLine(lines []string, start, end int, match func(string) bool, line string) []string {
	for i := start + 1; i < end; i++ {
		if match(lines[i]) {
			lines[i] = line
			return lines
		}
	}

	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:end]...)
	out = append(out, line)
	out = append(out, lines[end:]...)
	return out
}

// readRcLines 读取文件的文本行, 文件不存在时返回 nil
func readRcLines(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	text := strings.TrimSuffix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

// writeRcLines 写回文本行, 保留原有文件权限
func writeRcLines(filePath string, lines []string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(filePath); err == nil {
		mode = info.Mode().Perm()
	}

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(filePath, []byte(content), mode); err != nil {
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}
	return nil
}

// rcEnvLine 渲染托管块中的环境变量行, eg: export FOO='bar'
func rcEnvLine(name, value string) string {
	return "export " + name + "=" + rcQuote(value)
}

// rcEnvEntry 解析托管块中的环境变量行
func rcEnvEntry(line string) (name, value string, ok bool) {
	rest, ok := strings.CutPrefix(line, "export ")
	if !ok {
		return "", "", false
	}

	name, quoted, ok := strings.Cut(rest, "=")
	if !ok || name == "" {
		return "", "", false
	}

	value, ok = rcUnquote(quoted)
	if !ok {
		return "", "", false
	}
	return name, value, true
}

// matchRcEnvLine 返回匹配指定环境变量行的匹配函数
func matchRcEnvLine(name string) func(string) bool {
	prefix := "export " + name + "="
	return func(line string) bool {
		return strings.HasPrefix(line, prefix)
	}
}

// rcPathLine 渲染托管块中的 PATH 条目行, eg: export PATH='/opt/bin':$PATH
func rcPathLine(path string) string {
	return "export PATH=" + rcQuote(path) + ":$PATH"
}

// rcPathLinePrefix PATH 条目行的前缀
const rcPathLinePrefix = "export PATH="

// matchRcPathLine 返回匹配指定 PATH 条目行的匹配函数
func matchRcPathLine(path string) func(string) bool {
	return func(line string) bool {
		value, ok := rcPathValue(line)
		return ok && value == path
	}
}

// rcPathValue 解析托管块中的 PATH 条目行
func rcPathValue(line string) (string, bool) {
	rest, ok := strings.CutPrefix(line, rcPathLinePrefix)
	if !ok {
		return "", false
	}

	quoted, ok := strings.CutSuffix(rest, ":$PATH")
	if !ok {
		return "", false
	}
	return rcUnquote(quoted)
}

// rcQuote 使用单引号包裹 shell 值
func rcQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// rcUnquote 解析单引号包裹的 shell 值
func rcUnquote(quoted string) (string, bool) {
	if len(quoted) < 2 || quoted[0] != '\'' || quoted[len(quoted)-1] != '\'' {
		return "", false
	}
	return strings.ReplaceAll(quoted[1:len(quoted)-1], `'\''`, "'"), true
}
