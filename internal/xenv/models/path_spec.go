package models

import "strings"

var osPathPrefixes = map[string]struct{}{
	"windows": {},
	"linux":   {},
	"darwin":  {},
}

// FilterPathsForGOOS keeps common paths and paths prefixed for the target GOOS.
func FilterPathsForGOOS(paths []string, goos string) []string {
	if len(paths) == 0 {
		return nil
	}

	goos = strings.ToLower(goos)
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		prefix, value, ok := strings.Cut(path, ":")
		if !ok {
			filtered = append(filtered, path)
			continue
		}

		prefix = strings.ToLower(prefix)
		if _, supported := osPathPrefixes[prefix]; !supported {
			filtered = append(filtered, path)
			continue
		}
		if prefix == goos && value != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

// FilterSDKsForGOOS keeps common SDK versions and versions prefixed for the target GOOS.
func FilterSDKsForGOOS(sdks map[string]string, goos string) map[string]string {
	if len(sdks) == 0 {
		return nil
	}

	goos = strings.ToLower(goos)
	filtered := make(map[string]string, len(sdks))
	for name, version := range sdks {
		prefix, value, ok := strings.Cut(version, ":")
		if !ok {
			filtered[name] = version
			continue
		}

		prefix = strings.ToLower(prefix)
		if _, supported := osPathPrefixes[prefix]; !supported {
			filtered[name] = version
			continue
		}
		if prefix == goos && value != "" {
			filtered[name] = value
		}
	}
	return filtered
}
