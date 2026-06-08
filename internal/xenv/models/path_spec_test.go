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
