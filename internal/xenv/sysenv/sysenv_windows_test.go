//go:build windows

package sysenv

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"golang.org/x/sys/windows/registry"
)

// newTestKey 创建临时注册表键用于测试, 测试结束自动删除.
//
// NOTE: 只操作 HKCU\Software\xenv-test-*, 绝不触碰真实的 Environment 键
func newTestKey(t *testing.T) registry.Key {
	t.Helper()

	subKey := fmt.Sprintf(`Software\xenv-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	key, _, err := registry.CreateKey(registry.CURRENT_USER, subKey, registry.ALL_ACCESS)
	assert.NoErr(t, err)

	t.Cleanup(func() {
		assert.NoErr(t, key.Close())
		assert.NoErr(t, registry.DeleteKey(registry.CURRENT_USER, subKey))
	})
	return key
}

func TestSetVarInKeepsValueType(t *testing.T) {
	tests := []struct {
		name     string
		init     func(t *testing.T, key registry.Key)
		value    string
		wantType uint32
	}{
		{
			name:     "新建变量为 REG_SZ",
			value:    `D:\env\sdk\bin`,
			wantType: registry.SZ,
		},
		{
			name: "更新已有的 REG_EXPAND_SZ 保持类型且不展开",
			init: func(t *testing.T, key registry.Key) {
				assert.NoErr(t, key.SetExpandStringValue("XENV_VAR", `%USERPROFILE%\bin`))
			},
			value:    `%USERPROFILE%\tools\bin`,
			wantType: registry.EXPAND_SZ,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := newTestKey(t)
			if tt.init != nil {
				tt.init(t, key)
			}

			assert.NoErr(t, setVarIn(key, "XENV_VAR", tt.value))

			got, valType, err := key.GetStringValue("XENV_VAR")
			assert.NoErr(t, err)
			assert.Eq(t, tt.wantType, valType)
			assert.Eq(t, tt.value, got)
		})
	}
}

func TestUnsetVarIn(t *testing.T) {
	key := newTestKey(t)

	t.Run("删除已存在的变量", func(t *testing.T) {
		assert.NoErr(t, setVarIn(key, "XENV_VAR", "1"))
		assert.NoErr(t, unsetVarIn(key, "XENV_VAR"))

		_, _, err := key.GetStringValue("XENV_VAR")
		assert.ErrIs(t, err, registry.ErrNotExist)
	})

	t.Run("删除不存在的变量报错", func(t *testing.T) {
		err := unsetVarIn(key, "XENV_MISSING")
		assert.Err(t, err)
		assert.ErrMsgContains(t, err, "is not found in system environment")
	})
}

func TestAddPathInPrependsAndDedupes(t *testing.T) {
	key := newTestKey(t)

	// 空键时新建 PATH, 类型与系统默认一致
	assert.NoErr(t, addPathIn(key, `C:\bin`))
	value, valType, err := key.GetStringValue(pathVarName)
	assert.NoErr(t, err)
	assert.Eq(t, uint32(registry.EXPAND_SZ), valType)
	assert.Eq(t, `C:\bin`, value)

	// 后加入的排在前面(最高优先级)
	assert.NoErr(t, addPathIn(key, `D:\tools`))
	value, valType, err = key.GetStringValue(pathVarName)
	assert.NoErr(t, err)
	assert.Eq(t, uint32(registry.EXPAND_SZ), valType)
	assert.Eq(t, `D:\tools;C:\bin`, value)

	// 忽略大小写和分隔符差异的重复路径
	err = addPathIn(key, `c:/bin/`)
	assert.Err(t, err)
	assert.ErrMsgContains(t, err, "already exists in system PATH")

	got, err := pathListIn(key)
	assert.NoErr(t, err)
	assert.Eq(t, []string{`D:\tools`, `C:\bin`}, got)
}

func TestRemovePathIn(t *testing.T) {
	key := newTestKey(t)
	assert.NoErr(t, addPathIn(key, `C:\bin`))
	assert.NoErr(t, addPathIn(key, `D:\tools`))

	t.Run("删除已存在的路径", func(t *testing.T) {
		assert.NoErr(t, removePathIn(key, `c:\bin\`))

		got, err := pathListIn(key)
		assert.NoErr(t, err)
		assert.Eq(t, []string{`D:\tools`}, got)
	})

	t.Run("删除不存在的路径报错", func(t *testing.T) {
		err := removePathIn(key, `E:\nope`)
		assert.Err(t, err)
		assert.ErrMsgContains(t, err, "path not found in system PATH")
	})
}
