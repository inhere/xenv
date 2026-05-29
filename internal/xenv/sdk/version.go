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
func ParseMultipleVersionSpecs(specs []string) ([]*models.VersionSpec, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	var result []*VersionSpec
	for _, spec := range specs {
		parsed, err := ParseVersionSpec(spec)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}

	return result, nil
}

// IsValidSDKName 检查SDK名称是否有效
func IsValidSDKName(name string) bool {
	if name == "" {
		return false
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-') {
			return false
		}
	}

	return true
}

// IsValidVersion 检查版本号是否有效
func IsValidVersion(version string) bool {
	if version == "" {
		return false
	}

	invalidChars := []string{" ", "\t", "\n", "\r"}
	for _, char := range invalidChars {
		if strings.Contains(version, char) {
			return false
		}
	}

	return true
}

// NormalizeVersion 标准化版本号
func NormalizeVersion(version string) string {
	version = strings.TrimSpace(version)

	switch strings.ToLower(version) {
	case "lts":
		return "lts"
	case "latest":
		return "latest"
	case "stable":
		return "stable"
	case "auto":
		return "auto"
	default:
		return version
	}
}

// CompareVersions 比较两个版本号
func CompareVersions(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}
