package sdk

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestParseVersionSpec(t *testing.T) {
	testCases := []struct {
		input    string
		expected *VersionSpec
		hasError bool
	}{
		{
			input: "go:1.21.5",
			expected: &VersionSpec{
				Name:    "go",
				Version: "1.21.5",
			},
			hasError: false,
		},
		{
			input: "node:18",
			expected: &VersionSpec{
				Name:    "node",
				Version: "18",
			},
			hasError: false,
		},
		{
			input: "java:lts",
			expected: &VersionSpec{
				Name:    "java",
				Version: "lts",
			},
			hasError: false,
		},
		{
			input:    "",
			expected: nil,
			hasError: true,
		},
		{
			input: "go",
			expected: &VersionSpec{
				Name:    "go",
				Version: "latest",
			},
			hasError: false,
		},
		{
			input:    "go:",
			expected: nil,
			hasError: true,
		},
		{
			input:    ":1.21",
			expected: nil,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ParseVersionSpec(tc.input)

			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tc.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", tc.input, err)
				return
			}

			if result.Name != tc.expected.Name {
				t.Errorf("Expected Name %q, got %q", tc.expected.Name, result.Name)
			}

			if result.Version != tc.expected.Version {
				t.Errorf("Expected version %q, got %q", tc.expected.Version, result.Version)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	testCases := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{name: "same version", v1: "1.21.5", v2: "1.21.5", expected: 0},
		{name: "patch asc", v1: "1.21.4", v2: "1.21.5", expected: -1},
		{name: "patch desc", v1: "1.21.6", v2: "1.21.5", expected: 1},
		{name: "minor numeric order", v1: "1.9.0", v2: "1.10.0", expected: -1},
		{name: "minor numeric order desc", v1: "1.10.0", v2: "1.9.0", expected: 1},
		{name: "patch numeric order", v1: "1.26.9", v2: "1.26.10", expected: -1},
		{name: "patch numeric order desc", v1: "1.26.10", v2: "1.26.9", expected: 1},
		{name: "prerelease less than release", v1: "1.24.0-rc1", v2: "1.24.0", expected: -1},
		{name: "release greater than prerelease", v1: "1.24.0", v2: "1.24.0-rc1", expected: 1},
		{name: "prerelease numeric order", v1: "1.24.0-rc1", v2: "1.24.0-rc2", expected: -1},
		{name: "shorter version is smaller", v1: "18", v2: "18.1", expected: -1},
		{name: "non numeric fallback", v1: "go", v2: "node", expected: -1},
		{name: "non numeric fallback desc", v1: "node", v2: "go", expected: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Eq(t, tc.expected, CompareVersions(tc.v1, tc.v2))
		})
	}
}

func TestVersionSpecString(t *testing.T) {
	spec := &VersionSpec{
		Name:    "go",
		Version: "1.21.5",
	}

	expected := "go:1.21.5"
	result := spec.String()

	if result != expected {
		t.Errorf("VersionSpec.String() = %q, expected %q", result, expected)
	}
}
