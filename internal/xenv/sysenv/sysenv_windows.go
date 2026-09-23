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

	if err = setVarIn(key, name, value); err != nil {
		return err
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

	if err = unsetVarIn(key, name); err != nil {
		return err
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

	if err = addPathIn(key, path); err != nil {
		return err
	}

	notifyEnvChanged()
	return nil
}

// RemovePath 从用户级 PATH 中删除路径
func RemovePath(path string) error {
	key, err := openUserEnvKey()
	if err != nil {
		return err
	}
	defer key.Close()

	if err = removePathIn(key, path); err != nil {
		return err
	}

	notifyEnvChanged()
	return nil
}

// EnvVars 返回用户级环境变量. NOTE: 返回注册表中的原始值, 不展开 %VAR% 引用
func EnvVars() (map[string]string, error) {
	key, err := openUserEnvKey()
	if err != nil {
		return nil, err
	}
	defer key.Close()

	return envVarsIn(key)
}

// PathList 返回用户级 PATH 的条目
func PathList() ([]string, error) {
	key, err := openUserEnvKey()
	if err != nil {
		return nil, err
	}
	defer key.Close()

	return pathListIn(key)
}

// openUserEnvKey 打开当前用户的环境变量注册表键
func openUserEnvKey() (registry.Key, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, userEnvKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return 0, fmt.Errorf("failed to open user environment registry key: %w", err)
	}
	return key, nil
}

// setVarIn 在指定注册表键中设置环境变量
//
// 已存在的 REG_EXPAND_SZ 值会保持类型, 避免 %USERPROFILE% 之类的引用被展开
func setVarIn(key registry.Key, name, value string) error {
	var err error
	if isExpandString(key, name) {
		err = key.SetExpandStringValue(name, value)
	} else {
		err = key.SetStringValue(name, value)
	}
	if err != nil {
		return fmt.Errorf("failed to set system environment variable %s: %w", name, err)
	}
	return nil
}

// unsetVarIn 从指定注册表键中删除环境变量
func unsetVarIn(key registry.Key, name string) error {
	if _, _, err := key.GetStringValue(name); err != nil {
		if err == registry.ErrNotExist {
			return fmt.Errorf("environment variable %s is not found in system environment", name)
		}
		return fmt.Errorf("failed to read system environment variable %s: %w", name, err)
	}

	if err := key.DeleteValue(name); err != nil {
		return fmt.Errorf("failed to unset system environment variable %s: %w", name, err)
	}
	return nil
}

// addPathIn 在指定注册表键的 PATH 最前面插入路径
func addPathIn(key registry.Key, path string) error {
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

// removePathIn 从指定注册表键的 PATH 中删除路径
func removePathIn(key registry.Key, path string) error {
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

// envVarsIn 读取指定注册表键中的环境变量. NOTE: 返回原始值, 不展开 %VAR% 引用
func envVarsIn(key registry.Key) (map[string]string, error) {
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

// pathListIn 读取指定注册表键中 PATH 的条目
func pathListIn(key registry.Key) ([]string, error) {
	value, _, err := userPathValue(key)
	if err != nil {
		return nil, err
	}
	return splitPathList(value), nil
}

// isExpandString 检查已存在的值是否为 REG_EXPAND_SZ
func isExpandString(key registry.Key, name string) bool {
	_, valType, err := key.GetStringValue(name)
	return err == nil && valType == registry.EXPAND_SZ
}

// userPathValue 读取 PATH 的原始值和值类型. NOTE: GetStringValue 不会展开 %VAR%
func userPathValue(key registry.Key) (string, uint32, error) {
	value, valType, err := key.GetStringValue(pathVarName)
	if err == nil {
		return value, valType, nil
	}
	if err == registry.ErrNotExist {
		// 尚未设置 PATH, 新建时使用 REG_EXPAND_SZ, 与系统默认一致
		return "", registry.EXPAND_SZ, nil
	}
	return "", 0, fmt.Errorf("failed to read user PATH: %w", err)
}

// setUserPathValue 写回 PATH, 保持原有的值类型
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
