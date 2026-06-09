package models

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestFilterPathsForGOOS(t *testing.T) {
	cases := []struct {
		name  string
		goos  string
		paths []string
		want  []string
	}{
		{
			name: "windows keeps common and windows paths",
			goos: "windows",
			paths: []string{
				"./bin",
				"windows:C:/Program Files (x86)/NSIS",
				"linux:/opt/nsis/bin",
				"darwin:/opt/homebrew/bin",
			},
			want: []string{"./bin", "C:/Program Files (x86)/NSIS"},
		},
		{
			name: "linux keeps common and linux paths",
			goos: "linux",
			paths: []string{
				"./bin",
				"windows:C:/Program Files (x86)/NSIS",
				"linux:/opt/nsis/bin",
				"darwin:/opt/homebrew/bin",
			},
			want: []string{"./bin", "/opt/nsis/bin"},
		},
		{
			name:  "unknown prefixes stay as normal paths",
			goos:  "linux",
			paths: []string{"custom:/opt/tool/bin", "C:/tools/bin"},
			want:  []string{"custom:/opt/tool/bin", "C:/tools/bin"},
		},
		{
			name:  "os prefixes are case insensitive and empty values are skipped",
			goos:  "windows",
			paths: []string{"Windows:C:/tools/bin", "windows:"},
			want:  []string{"C:/tools/bin"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterPathsForGOOS(tt.paths, tt.goos)
			assert.Eq(t, tt.want, got)
		})
	}
}

func TestFilterSDKsForGOOS(t *testing.T) {
	cases := []struct {
		name string
		goos string
		sdks map[string]string
		want map[string]string
	}{
		{
			name: "windows keeps common and windows sdks",
			goos: "windows",
			sdks: map[string]string{
				"go":      "1.24",
				"flutter": "windows:3.27",
				"node":    "linux:20",
				"brew":    "darwin:4.0",
			},
			want: map[string]string{
				"go":      "1.24",
				"flutter": "3.27",
			},
		},
		{
			name: "linux keeps common and linux sdks",
			goos: "linux",
			sdks: map[string]string{
				"go":      "1.24",
				"flutter": "windows:3.27",
				"node":    "linux:20",
			},
			want: map[string]string{
				"go":   "1.24",
				"node": "20",
			},
		},
		{
			name: "unknown prefixes stay as normal versions",
			goos: "linux",
			sdks: map[string]string{
				"tool": "custom:1.0",
			},
			want: map[string]string{
				"tool": "custom:1.0",
			},
		},
		{
			name: "os prefixes are case insensitive and empty values are skipped",
			goos: "windows",
			sdks: map[string]string{
				"flutter": "Windows:3.27",
				"empty":   "windows:",
			},
			want: map[string]string{
				"flutter": "3.27",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterSDKsForGOOS(tt.sdks, tt.goos)
			assert.Eq(t, tt.want, got)
		})
	}
}
