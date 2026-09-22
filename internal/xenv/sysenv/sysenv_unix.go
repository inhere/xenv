//go:build !windows

package sysenv

import (
	"fmt"
	"os"

	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

// SetVar 设置用户级环境变量, 写入当前 shell 的用户启动文件
func SetVar(name, value string) error {
	filePath, err := rcFilePath()
	if err != nil {
		return err
	}
	return upsertBlockLine(filePath, matchRcEnvLine(name), rcEnvLine(name, value))
}

// UnsetVar 删除用户启动文件中设置的环境变量
func UnsetVar(name string) error {
	filePath, err := rcFilePath()
	if err != nil {
		return err
	}

	found, err := removeBlockLine(filePath, matchRcEnvLine(name))
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("environment variable %s is not found in system environment", name)
	}
	return nil
}

// AddPath 添加路径到用户启动文件中的 PATH 最前面(最高优先级)
func AddPath(path string) error {
	filePath, err := rcFilePath()
	if err != nil {
		return err
	}

	match := matchRcPathLine(path)
	exists, err := blockLineExists(filePath, match)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("path already exists in system PATH: %s", path)
	}
	return upsertBlockLine(filePath, match, rcPathLine(path))
}

// RemovePath 删除用户启动文件中的 PATH 条目
func RemovePath(path string) error {
	filePath, err := rcFilePath()
	if err != nil {
		return err
	}

	found, err := removeBlockLine(filePath, matchRcPathLine(path))
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("path not found in system PATH: %s", path)
	}
	return nil
}

// EnvVars 返回用户启动文件中 xenv 托管块设置的环境变量
func EnvVars() (map[string]string, error) {
	filePath, err := rcFilePath()
	if err != nil {
		return nil, err
	}

	lines, err := blockContent(filePath)
	if err != nil {
		return nil, err
	}
	return blockEnvVars(lines), nil
}

// PathList 返回用户启动文件中 xenv 托管块添加的 PATH 条目
func PathList() ([]string, error) {
	filePath, err := rcFilePath()
	if err != nil {
		return nil, err
	}

	lines, err := blockContent(filePath)
	if err != nil {
		return nil, err
	}
	return blockPaths(lines), nil
}

// rcFilePath 返回需要写入的用户 shell 启动文件
func rcFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home dir: %w", err)
	}

	shellName := xenvcom.HookShell()
	if shellName == "" {
		shellName = os.Getenv("SHELL")
	}
	return rcFileFor(shellName, homeDir)
}
