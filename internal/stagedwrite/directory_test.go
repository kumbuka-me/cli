package stagedwrite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceDirectoryInstallsPreparedOutput(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	target := filepath.Join(parent, "site")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644))
	err := ReplaceDirectory(target, func(staging string) error {
		data, err := os.ReadFile(filepath.Join(target, "old.txt"))
		require.NoError(t, err)
		assert.Equal(t, "old", string(data), "existing output remains available during population")
		return os.WriteFile(filepath.Join(staging, "new.txt"), []byte("new"), 0o644)
	})
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(target, "new.txt"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
	assert.NoFileExists(t, filepath.Join(target, "old.txt"))
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Len(t, entries, 1, "temporary staging and backup directories must be removed")
	assert.Equal(t, "site", entries[0].Name())
}

func TestReplaceDirectoryPreservesExistingOutputWhenPopulateFails(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	target := filepath.Join(parent, "site")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "current.txt"), []byte("current"), 0o644))
	failure := errors.New("generation failed")
	err := ReplaceDirectory(target, func(staging string) error {
		require.NoError(t, os.WriteFile(filepath.Join(staging, "partial.txt"), []byte("partial"), 0o644))
		return failure
	})
	require.ErrorIs(t, err, failure)
	data, err := os.ReadFile(filepath.Join(target, "current.txt"))
	require.NoError(t, err)
	assert.Equal(t, "current", string(data))
	assert.NoFileExists(t, filepath.Join(target, "partial.txt"))
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "site", entries[0].Name())
}

func TestReplaceDirectoryCreatesMissingParents(t *testing.T) {
	t.Parallel()
	target := filepath.Join(t.TempDir(), "nested", "site")
	require.NoError(t, ReplaceDirectory(target, func(staging string) error {
		return os.WriteFile(filepath.Join(staging, "new.txt"), []byte("new"), 0o644)
	}))
	data, err := os.ReadFile(filepath.Join(target, "new.txt"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}

func TestReplaceDirectoryRejectsFileTarget(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	target := filepath.Join(parent, "site")
	require.NoError(t, os.WriteFile(target, []byte("keep"), 0o644))
	err := ReplaceDirectory(target, func(staging string) error {
		return os.WriteFile(filepath.Join(staging, "new.txt"), []byte("new"), 0o644)
	})
	require.ErrorContains(t, err, "is not a directory")
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(data))
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestReplaceDirectoryRejectsFileParent(t *testing.T) {
	t.Parallel()
	parent := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(parent, []byte("keep"), 0o644))
	called := false
	err := ReplaceDirectory(filepath.Join(parent, "site"), func(string) error { called = true; return nil })
	require.ErrorContains(t, err, "create output parent")
	assert.False(t, called)
	data, err := os.ReadFile(parent)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(data))
}

func TestReplaceDirectoryRestoresOutputWhenInstallFails(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	target := filepath.Join(parent, "site")
	require.NoError(t, os.Mkdir(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "current.txt"), []byte("current"), 0o644))
	err := ReplaceDirectory(target, func(staging string) error {
		// Removing the prepared directory deterministically forces the final rename to fail.
		return os.RemoveAll(staging)
	})
	require.ErrorContains(t, err, "install staged output")
	assert.ErrorIs(t, err, os.ErrNotExist)
	data, err := os.ReadFile(filepath.Join(target, "current.txt"))
	require.NoError(t, err)
	assert.Equal(t, "current", string(data))
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "site", entries[0].Name())
}
