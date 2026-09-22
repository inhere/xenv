package sysenv

import (
	"strings"
)

// winPathListSep PATH 变量值中条目的分隔符
const winPathListSep = ";"

// splitPathList 拆分 PATH 变量值, 忽略空白条目
func splitPathList(value string) []string {
	var pathList []string
	for _, path := range strings.Split(value, winPathListSep) {
		if path = strings.TrimSpace(path); path != "" {
			pathList = append(pathList, path)
		}
	}
	return pathList
}

// pathListIndex 查找 PATH 条目位置, 按 Windows 规则忽略大小写和冗余分隔符
//
// return 未找到时返回 -1
func pathListIndex(pathList []string, path string) int {
	target := normalizeWinPath(path)
	for i, item := range pathList {
		if strings.EqualFold(normalizeWinPath(item), target) {
			return i
		}
	}
	return -1
}

// normalizeWinPath 统一 Windows 路径分隔符并去掉末尾分隔符, 便于比较
//
// 不展开 %USERPROFILE% 之类的变量, 也不解析 . / .. 段, 避免改动用户已有的 PATH 条目
func normalizeWinPath(path string) string {
	return strings.TrimRight(strings.ReplaceAll(path, "/", `\`), `\`)
}
