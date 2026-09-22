//go:build windows

package sysenv

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// userEnvKeyPath 用户级环境变量所在的注册表子键
const userEnvKeyPath = `Environment`

// pathVarName PATH 变量在注册表中的值名称
const pathVarName = `Path`

// SetVar 设置用户级环境变量, 保存到 HKCU\Environment
//
// 已存在的 REG_EXPAND_SZ 值会保持类型, 避免 %USERPROFILE% 之类的引用被展开
func SetVar(name, value string) error {
	key, err := openUserEnvKey()
	if err != nil {
		return err
	}
	defer key.Close()

	if isExpandString(key, name) {
		err = key.SetExpandStringValue(name, value)
	} else {
		err = key.SetStringValue(name, value)
	}
	if err != nil {
		return fmt.Errorf("failed to set system environment variable %s: %w", name, err)
	}

	notifyEnvChanged()
	return nil
}

// UnsetVar 删除用户级环境变量
func UnsetVar(name string) error {
	key, err := openUserEnvKey()
	if err != nil {
		return err
	}
	defer key.Close()

	if _, _, err = key.GetStringValue(name); err != nil {
		if err == registry.ErrNotExist {
			return fmt.Errorf("environment variable %s is not found in system environment", name)
		}
		return fmt.Errorf("failed to read system environment variable %s: %w", name, err)
	}

	if err = key.DeleteValue(name); err != nil {
		return fmt.Errorf("failed to unset system environment variable %s: %w", name, err)
	}

	notifyEnvChanged()
	return nil
}

// AddPath 添加路径到用户级 PATH 的最前面(最高优先级)
func AddPath(path string) error {
	key, err := openUserEnvKey()
	if err != nil {
		return err
	}
	defer key.Close()

	value, valType, err := userPathValue(key)
	if err != nil {
		return err
	}

	pathList := splitPathList(value)
	if pathListIndex(pathList, path) >= 0 {
		return fmt.Errorf("path already exists in system PATH: %s", path)
	}

	pathList = append([]string{path}, pathList...)
	return setUserPathValue(key, strings.Join(pathList, winPathListSep), valType)
}

// RemovePath 从用户级 PATH 中删除路径
func RemovePath(path string) error {
	key, err := openUserEnvKey()
	if err != nil {
		return err
	}
	defer key.Close()

	value, valType, err := userPathValue(key)
	if err != nil {
		return err
	}

	pathList := splitPathList(value)
	idx := pathListIndex(pathList, path)
	if idx < 0 {
		return fmt.Errorf("path not found in system PATH: %s", path)
	}

	pathList = append(pathList[:idx], pathList[idx+1:]...)
	return setUserPathValue(key, strings.Join(pathList, winPathListSep), valType)
}

// EnvVars 返回用户级环境变量. NOTE: 返回注册表中的原始值, 不展开 %VAR% 引用
func EnvVars() (map[string]string, error) {
	key, err := openUserEnvKey()
	if err != nil {
		return nil, err
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		return nil, fmt.Errorf("failed to read user environment values: %w", err)
	}

	envs := make(map[string]string, len(names))
	for _, name := range names {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			// 跳过 DWORD 等非字符串类型的值
			if err == registry.ErrUnexpectedType {
				continue
			}
			return nil, fmt.Errorf("failed to read user environment value %s: %w", name, err)
		}
		envs[name] = value
	}
	return envs, nil
}

// PathList 返回用户级 PATH 的条目
func PathList() ([]string, error) {
	key, err := openUserEnvKey()
	if err != nil {
		return nil, err
	}
	defer key.Close()

	value, _, err := userPathValue(key)
	if err != nil {
		return nil, err
	}
	return splitPathList(value), nil
}

// openUserEnvKey 打开当前用户的环境变量注册表键
func openUserEnvKey() (registry.Key, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, userEnvKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return 0, fmt.Errorf("failed to open user environment registry key: %w", err)
	}
	return key, nil
}

// isExpandString 检查已存在的值是否为 REG_EXPAND_SZ
func isExpandString(key registry.Key, name string) bool {
	_, valType, err := key.GetStringValue(name)
	return err == nil && valType == registry.EXPAND_SZ
}

// userPathValue 读取用户 PATH 的原始值和值类型. NOTE: GetStringValue 不会展开 %VAR%
func userPathValue(key registry.Key) (string, uint32, error) {
	value, valType, err := key.GetStringValue(pathVarName)
	if err == nil {
		return value, valType, nil
	}
	if err == registry.ErrNotExist {
		// 尚未设置用户 PATH, 新建时使用 REG_EXPAND_SZ, 与系统默认一致
		return "", registry.EXPAND_SZ, nil
	}
	return "", 0, fmt.Errorf("failed to read user PATH: %w", err)
}

// setUserPathValue 写回用户 PATH, 保持原有的值类型
func setUserPathValue(key registry.Key, value string, valType uint32) error {
	var err error
	if valType == registry.EXPAND_SZ {
		err = key.SetExpandStringValue(pathVarName, value)
	} else {
		err = key.SetStringValue(pathVarName, value)
	}
	if err != nil {
		return fmt.Errorf("failed to update user PATH: %w", err)
	}

	notifyEnvChanged()
	return nil
}

// notifyEnvChanged 广播 WM_SETTINGCHANGE, 让新的进程和资源管理器读取更新后的用户环境.
//
// 广播失败不影响已经写入的注册表内容, 因此忽略返回结果
func notifyEnvChanged() {
	envName, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}

	// SendMessageTimeoutW(HWND_BROADCAST, WM_SETTINGCHANGE, 0, "Environment", SMTO_ABORTIFHUNG, 5000, nil)
	proc := windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
	_, _, _ = proc.Call(uintptr(0xffff), uintptr(0x001A), 0, uintptr(unsafe.Pointer(envName)), uintptr(0x0002), 5000, 0)
}
