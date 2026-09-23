// Package pathutil provides filesystem path identity helpers shared by CLI subsystems.
package pathutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ResolveExisting returns an absolute path with every existing prefix resolved through symlinks.
func ResolveExisting(filename string) (string, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}

	volume := filepath.VolumeName(absolute)
	remainder := strings.TrimPrefix(absolute, volume)
	current := volume + string(filepath.Separator)
	remainder = strings.TrimPrefix(remainder, string(filepath.Separator))
	parts := strings.Split(remainder, string(filepath.Separator))

	for index, part := range parts {
		if part == "" {
			continue
		}

		candidate := filepath.Join(current, part)
		info, err := os.Lstat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			return appendPathSuffix(candidate, parts[index+1:]), nil
		}
		if err != nil {
			return "", err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			current, err = filepath.EvalSymlinks(candidate)
			if err != nil {
				return "", err
			}
			info, err = os.Stat(current)
			if err != nil {
				return "", err
			}
		} else {
			current = candidate
		}

		if !info.IsDir() && hasPathSuffix(parts[index+1:]) {
			return appendPathSuffix(current, parts[index+1:]), nil
		}
	}

	return filepath.Clean(current), nil
}

// appendPathSuffix appends unresolved path components to one resolved prefix.
func appendPathSuffix(prefix string, parts []string) string {
	for _, part := range parts {
		if part != "" {
			prefix = filepath.Join(prefix, part)
		}
	}
	return filepath.Clean(prefix)
}

// hasPathSuffix reports whether unresolved path components remain.
func hasPathSuffix(parts []string) bool {
	for _, part := range parts {
		if part != "" {
			return true
		}
	}
	return false
}

// Contains reports whether child is equal to or nested below parent.
func Contains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
