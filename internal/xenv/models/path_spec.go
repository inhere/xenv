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
