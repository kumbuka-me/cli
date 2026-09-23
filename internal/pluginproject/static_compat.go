package pluginproject

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/kumbuka-me/sdk/pluginpackage"
)

// staticCompatibleArchive rewrites admin-only configuration fields unsupported by older static runtimes.
func staticCompatibleArchive(archive []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}

	files, err := readStaticArchive(reader)
	if err != nil {
		return nil, err
	}

	manifest, ok := files["plugin.yaml"]
	if !ok {
		return archive, nil
	}

	compatible := staticCompatibleManifest(string(manifest))
	if compatible == string(manifest) {
		return archive, nil
	}

	files["plugin.yaml"] = []byte(compatible)
	return writeStaticArchive(files)
}

// readStaticArchive validates and expands one bounded plugin archive into regular files.
func readStaticArchive(reader *zip.Reader) (map[string][]byte, error) {
	if len(reader.File) > pluginpackage.MaxFiles {
		return nil, fmt.Errorf("too many plugin archive entries")
	}

	files := make(map[string][]byte, len(reader.File))
	seen := make(map[string]bool, len(reader.File))
	total := 0
	for _, file := range reader.File {
		name := strings.TrimSuffix(file.Name, "/")
		if err := validateStaticArchiveEntry(file, name, seen); err != nil {
			return nil, err
		}
		seen[name] = true

		if file.FileInfo().IsDir() {
			continue
		}

		content, err := readStaticArchiveFile(file, pluginpackage.MaxExpandedBytes-total)
		if err != nil {
			return nil, err
		}
		total += len(content)
		files[name] = content
	}

	return files, nil
}

// validateStaticArchiveEntry checks one archive entry path, type, and directory shape.
func validateStaticArchiveEntry(file *zip.File, name string, seen map[string]bool) error {
	if !validStaticPackagePath(name) || seen[name] {
		return fmt.Errorf("invalid or duplicate plugin path %q", file.Name)
	}

	if kind := file.Mode().Type(); kind != 0 && kind != fs.ModeDir {
		return fmt.Errorf("plugin entry %q is not a regular file or directory", name)
	}

	if file.FileInfo().IsDir() {
		return validateStaticArchiveDirectory(file, name)
	}

	return nil
}

// validateStaticArchiveDirectory checks the archive shape and namespace of one directory entry.
func validateStaticArchiveDirectory(file *zip.File, name string) error {
	if !strings.HasSuffix(file.Name, "/") || file.UncompressedSize64 != 0 {
		return fmt.Errorf("invalid plugin directory %q", file.Name)
	}
	if name != "assets" && !strings.HasPrefix(name, "assets/") {
		return fmt.Errorf("unsupported plugin directory %q", name)
	}

	return nil
}

// readStaticArchiveFile reads one regular entry without exceeding the remaining expanded-byte budget.
func readStaticArchiveFile(file *zip.File, remaining int) ([]byte, error) {
	if remaining < 0 || file.UncompressedSize64 > uint64(remaining) {
		return nil, fmt.Errorf("plugin archive exceeds expanded size limit")
	}

	entry, err := file.Open()
	if err != nil {
		return nil, err
	}

	content, readErr := io.ReadAll(io.LimitReader(entry, int64(remaining)+1))
	closeErr := entry.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(content) > remaining {
		return nil, fmt.Errorf("plugin archive exceeds expanded size limit")
	}

	return content, nil
}

// staticCompatibleManifest removes administrator-only list schema details from a static runtime manifest.
func staticCompatibleManifest(source string) string {
	lines := strings.Split(source, "\n")
	result := make([]string, 0, len(lines))

	for index := 0; index < len(lines); {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		indent := leadingManifestSpaces(line)
		switch trimmed {
		case "type: color":
			result = append(result, strings.Repeat(" ", indent)+"type: text")
			index++
			continue
		case "type: list":
			result = append(result, strings.Repeat(" ", indent)+"type: textarea")
			index++
			for index < len(lines) {
				candidate := lines[index]
				candidateTrimmed := strings.TrimSpace(candidate)
				candidateIndent := leadingManifestSpaces(candidate)
				if startsNextConfigurationField(candidateTrimmed, candidateIndent, indent) {
					break
				}
				if candidateIndent == indent && strings.HasPrefix(candidateTrimmed, "max_items:") {
					index++
					continue
				}
				if candidateIndent == indent && candidateTrimmed == "columns:" {
					index++
					for index < len(lines) && leadingManifestSpaces(lines[index]) > indent {
						index++
					}
					continue
				}
				result = append(result, candidate)
				index++
			}
			continue
		default:
			result = append(result, line)
			index++
		}
	}

	return strings.Join(result, "\n")
}

// startsNextConfigurationField reports whether a manifest line leaves the current list-field schema.
func startsNextConfigurationField(trimmed string, candidateIndent, fieldIndent int) bool {
	if trimmed == "" {
		return false
	}
	if candidateIndent < fieldIndent {
		return true
	}

	return candidateIndent == fieldIndent && strings.HasPrefix(trimmed, "- id:")
}

// leadingManifestSpaces returns the number of leading ASCII spaces in a manifest line.
func leadingManifestSpaces(value string) int {
	count := 0
	for count < len(value) && value[count] == ' ' {
		count++
	}
	return count
}

// validStaticPackagePath reports whether an archive path is canonical and relative.
func validStaticPackagePath(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	if name == "." || name == ".." || strings.HasPrefix(name, "../") {
		return false
	}

	return path.Clean(name) == name
}

// writeStaticArchive serializes rewritten static package files with deterministic metadata.
func writeStaticArchive(files map[string][]byte) ([]byte, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write(files[name]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
