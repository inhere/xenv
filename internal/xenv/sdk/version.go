package sdk

import (
	"errors"
	"strings"

	"github.com/inhere/xenv/internal/xenv/models"
)

var (
	ErrInvalidVersionSpec = errors.New("invalid version specification")
	ErrEmptyVersionSpec   = errors.New("empty version specification")
)

// VersionSpec 版本规格
type VersionSpec = models.VersionSpec

// ParseVersionSpec 解析版本规格 format: "sdk" or "sdk:version" or "sdk@version"
func ParseVersionSpec(spec string) (*models.VersionSpec, error) {
	if spec == "" {
		return nil, ErrEmptyVersionSpec
	}

	sep := ":"
	if strings.Contains(spec, "@") {
		sep = "@"
	}

	parts := strings.SplitN(spec, sep, 2)
	if len(parts) != 2 {
		parts = append(parts, "latest")
	}

	sdk := strings.TrimSpace(parts[0])
	version := strings.TrimSpace(parts[1])
	if sdk == "" || version == "" {
		return nil, ErrInvalidVersionSpec
	}

	return &VersionSpec{
		Name:    sdk,
		Version: version,
	}, nil
}

// ParseMultipleVersionSpecs 解析多个版本规格
// IsValidSDKName 检查SDK名称是否有效
// IsValidVersion 检查版本号是否有效
// NormalizeVersion 标准化版本号
// CompareVersions 比较两个版本号，返回 -1/0/1。
// 与 models.CompareVersionStrings 规则一致：数字段按数值比较，预发布版本小于正式版本。
func CompareVersions(v1, v2 string) int {
	return models.CompareVersionStrings(v1, v2)
}
