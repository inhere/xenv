package models

import (
	"sort"
	"strings"
	"time"

	"github.com/inhere/xenv/internal/util"
)

type SDKLocalIndex struct {
	Schema    int            `json:"schema"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	SDKs      []InstalledSDK `json:"sdks"`
}

type InstalledSDK struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Version    string     `json:"version"`
	InstallDir string     `json:"install_dir"`
	Source     string     `json:"source"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Index      int        `json:"-"`
	Config     *ToolChain `json:"-"`
}

func NewSDKLocalIndex() *SDKLocalIndex {
	return &SDKLocalIndex{Schema: 1}
}

func (idx *SDKLocalIndex) ListByName(name string) []InstalledSDK {
	var items []InstalledSDK
	for i, sdk := range idx.SDKs {
		if sdk.Name == name {
			sdk.Index = i
			items = append(items, sdk)
		}
	}
	// 按版本降序（新 -> 旧）
	SortByVersionDesc(items)
	return items
}

// SortByVersionDesc 按版本号降序（新 -> 旧）排序，同版本保持原有相对顺序。
func SortByVersionDesc(items []InstalledSDK) {
	sort.SliceStable(items, func(i, j int) bool {
		return CompareVersionStrings(items[i].Version, items[j].Version) > 0
	})
}

// CompareVersionStrings 比较两个版本号，返回 -1/0/1。
// 规则：按 '.' 与 '-' 切段，数字段按数值比较（如 1.9.0 < 1.10.0）；
// 正式段相同时，带预发布后缀的版本更小（如 1.24.0-rc1 < 1.24.0）；非数字段退化为字典序。
func CompareVersionStrings(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}

	core1, pre1, hasPre1 := splitVersionSuffix(v1)
	core2, pre2, hasPre2 := splitVersionSuffix(v2)

	if cmp := compareVersionParts(core1, core2); cmp != 0 {
		return cmp
	}

	// 正式段相同：有预发布后缀的版本小于正式版
	if hasPre1 != hasPre2 {
		if hasPre1 {
			return -1
		}
		return 1
	}

	return compareVersionParts(pre1, pre2)
}

// splitVersionSuffix 以第一个 '-' 将版本拆为正式段与预发布段
func splitVersionSuffix(version string) (core, pre string, hasPre bool) {
	if idx := strings.IndexByte(version, '-'); idx >= 0 {
		return version[:idx], version[idx+1:], true
	}
	return version, "", false
}

// compareVersionParts 按 '.'/'-' 切段逐段比较，缺失段按 0 处理
func compareVersionParts(a, b string) int {
	if a == b {
		return 0
	}

	segsA := strings.FieldsFunc(a, isVersionSep)
	segsB := strings.FieldsFunc(b, isVersionSep)

	for i := range max(len(segsA), len(segsB)) {
		if cmp := compareVersionSegment(versionSegAt(segsA, i), versionSegAt(segsB, i)); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func isVersionSep(r rune) bool {
	return r == '.' || r == '-'
}

func versionSegAt(segs []string, i int) string {
	if i < len(segs) {
		return segs[i]
	}
	return "0"
}

// compareVersionSegment 数字段按数值比较，其余按字典序
func compareVersionSegment(a, b string) int {
	numA, okA := trimVersionNumber(a)
	numB, okB := trimVersionNumber(b)

	if okA && okB {
		// 去掉前导零后，长度不同则长者更大；长度相同字典序即数值序
		if len(numA) != len(numB) {
			if len(numA) < len(numB) {
				return -1
			}
			return 1
		}
		return strings.Compare(numA, numB)
	}

	return strings.Compare(a, b)
}

// trimVersionNumber 去掉数字段前导零；纯数字（含空串，按 0）返回 ok=true
func trimVersionNumber(seg string) (string, bool) {
	if seg == "" {
		return "0", true
	}

	for i := range len(seg) {
		if seg[i] < '0' || seg[i] > '9' {
			return seg, false
		}
	}

	trimmed := strings.TrimLeft(seg, "0")
	if trimmed == "" {
		return "0", true
	}
	return trimmed, true
}

func (idx *SDKLocalIndex) FindByID(id string) *InstalledSDK {
	for i := range idx.SDKs {
		sdk := &idx.SDKs[i]
		if sdk.ID == id {
			sdk.Index = i
			return sdk
		}
	}
	return nil
}

func (s *InstalledSDK) BinDirPath() string {
	if s.Config == nil {
		return util.NormalizePath(s.InstallDir)
	}
	return s.Config.FullBinPath(s.InstallDir)
}

func (s *InstalledSDK) RenderActiveEnv() map[string]string {
	if s.Config == nil || len(s.Config.ActiveEnv) == 0 {
		return nil
	}
	return s.Config.RenderActiveEnv(map[string]string{
		"name":        s.Name,
		"version":     s.Version,
		"install_dir": util.NormalizePath(s.InstallDir),
	})
}
