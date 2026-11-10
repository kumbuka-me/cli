package preview

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kumbuka-me/cli/internal/site"
)

const (
	watchInterval = 300 * time.Millisecond
	watchDebounce = 350 * time.Millisecond
)

// WatchPaths lists source files and directories whose changes require a site rebuild.
func WatchPaths(config site.Config, configPath string) []string {
	return []string{
		configPath,
		config.SourceDir,
		config.AssetsDir,
		config.Logo,
		config.Favicon,
		config.FaviconICO,
		config.PluginsFile,
	}
}

// fileState captures enough filesystem metadata to detect edits without reading file contents.
type fileState struct {
	// mode records file type and permissions.
	mode fs.FileMode
	// modTime is the last modification time in nanoseconds.
	modTime int64
	// size is the file size in bytes.
	size int64
	// missing indicates a watched file that does not yet exist.
	missing bool
}

// snapshot records file state recursively for all configured watch paths.
func snapshot(paths []string) (map[string]fileState, error) {
	result := make(map[string]fileState)
	for _, root := range paths {
		if root == "" {
			continue
		}
		absolute, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(absolute, func(name string, entry fs.DirEntry, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) {
				result[name] = fileState{missing: true}
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if name != absolute && entry.IsDir() && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			result[name] = fileState{mode: info.Mode(), modTime: info.ModTime().UnixNano(), size: info.Size()}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Watch polls input files, debounces changes, and calls rebuild until the context is canceled.
// Paths is evaluated again after each rebuild so changed site configurations can change watched directories.
func Watch(ctx context.Context, paths func() []string, rebuild func(), onError func(error)) {
	watch(ctx, paths, rebuild, onError, watchInterval, watchDebounce)
}

// watch implements the poll/debounce loop with injectable timing for deterministic tests.
func watch(ctx context.Context, paths func() []string, rebuild func(), onError func(error), interval, debounce time.Duration) {
	previous, err := snapshot(paths())
	if err != nil {
		onError(err)
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var changedAt time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		current, err := snapshot(paths())
		if err != nil {
			onError(err)
			continue
		}
		if !sameSnapshot(current, previous) {
			changedAt = time.Now()
			previous = current
		}
		if changedAt.IsZero() || time.Since(changedAt) < debounce {
			continue
		}
		changedAt = time.Time{}
		beforePaths := paths()
		rebuild()
		afterPaths := paths()
		updated, err := snapshot(afterPaths)
		if err != nil {
			onError(err)
			return
		}
		// If a file changes during the build, queue another rebuild instead of
		// silently treating that edit as already rendered.
		if slices.Equal(beforePaths, afterPaths) && !sameSnapshot(previous, updated) {
			changedAt = time.Now()
		}
		previous = updated
	}
}

// sameSnapshot checks whether watched paths retain identical filesystem metadata.
func sameSnapshot(a, b map[string]fileState) bool {
	if len(a) != len(b) {
		return false
	}
	for filename, state := range a {
		if previous, found := b[filename]; !found || previous != state {
			return false
		}
	}
	return true
}
