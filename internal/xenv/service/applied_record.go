package service

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv/models"
	"github.com/inhere/xenv/internal/xenv/shell"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

// recordDelta 描述一次命令给当前 shell 带来的可撤销变更
type recordDelta struct {
	// AddPaths 新加入 shell PATH 的条目
	AddPaths []string
	// RemPaths 从 shell PATH 移除的条目
	RemPaths []string
	// EnvNames 被修改的环境变量名, 应用前的值在合并记录时读取
	EnvNames []string
	// SDKs 本次激活的 SDK name => version
	SDKs map[string]string
}

// loadAppliedRecord 读取当前 shell 的 direnv 应用记录
//
// 记录由应用脚本写入 XENV_APPLIED_DIRENV, 内容损坏时视为无记录
func loadAppliedRecord() *models.AppliedDirenv {
	raw := os.Getenv(xenvcom.AppliedDirenvEnvName)
	if raw == "" {
		return nil
	}

	rec := &models.AppliedDirenv{}
	if err := json.Unmarshal([]byte(raw), rec); err != nil {
		ccolor.Warnf("WARN: invalid %s, ignore the previous direnv record: %v\n", xenvcom.AppliedDirenvEnvName, err)
		return nil
	}
	if rec.SDKs == nil {
		rec.SDKs = make(map[string]string)
	}
	return rec
}

// writeAppliedRecord 生成写入或清除 direnv 应用记录的脚本
func writeAppliedRecord(gen *shell.XenvScriptGenerator, rec *models.AppliedDirenv) string {
	if rec == nil || rec.IsEmpty() {
		return gen.GenUnsetEnv(xenvcom.AppliedDirenvEnvName)
	}

	data, err := json.Marshal(rec)
	if err != nil {
		ccolor.Warnf("WARN: failed to encode direnv record: %v\n", err)
		return gen.GenUnsetEnv(xenvcom.AppliedDirenvEnvName)
	}
	return gen.GenSetEnv(xenvcom.AppliedDirenvEnvName, string(data))
}

// mergeAppliedRecord 把本次命令的变更合并进当前记录, 并生成写入记录的脚本
//
// 无记录时创建只含本次变更的记录, 其 file 为空, 避免 SetupDirenv 误判为已应用
func mergeAppliedRecord(gen *shell.XenvScriptGenerator, delta recordDelta) string {
	if gen == nil {
		return ""
	}

	rec := loadAppliedRecord()
	if rec == nil {
		rec = models.NewAppliedDirenv("")
	}

	for _, path := range delta.AddPaths {
		rec.AddAppliedPath(path)
	}
	for _, path := range delta.RemPaths {
		rec.RemoveAppliedPath(path)
	}

	envNames := append([]string(nil), delta.EnvNames...)
	sort.Strings(envNames)
	for _, name := range envNames {
		prev, had := os.LookupEnv(name)
		rec.AddAppliedEnv(name, prev, had)
	}

	for name, version := range delta.SDKs {
		rec.SDKs[name] = version
	}
	return writeAppliedRecord(gen, rec)
}

// deltaFromActivate 把 SDK 激活参数转换为可撤销变更
func deltaFromActivate(params *models.ActivateSDKsParams) recordDelta {
	delta := recordDelta{SDKs: params.AddSdks}
	delta.AddPaths = append(delta.AddPaths, params.AddPaths...)
	delta.RemPaths = append(delta.RemPaths, params.RemPaths...)

	for name := range params.AddEnvs {
		delta.EnvNames = append(delta.EnvNames, name)
	}
	return delta
}
