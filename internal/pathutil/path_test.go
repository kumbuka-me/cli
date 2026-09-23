package pathutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveExisting(t *testing.T) {
	t.Parallel()

	t.Run("resolves symlinked existing prefix", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		realDirectory := filepath.Join(root, "real")
		require.NoError(t, os.Mkdir(realDirectory, 0o755))
		link := filepath.Join(root, "link")
		if err := os.Symlink(realDirectory, link); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}

		resolved, err := ResolveExisting(filepath.Join(link, "missing", "file"))

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(realDirectory, "missing", "file"), resolved)
	})

	t.Run("retains suffix below regular file", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		filename := filepath.Join(root, "file")
		require.NoError(t, os.WriteFile(filename, []byte("content"), 0o644))

		resolved, err := ResolveExisting(filepath.Join(filename, "child"))

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(filename, "child"), resolved)
	})
}

func TestContains(t *testing.T) {
	t.Parallel()

	t.Run("accepts same path", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		assert.True(t, Contains(root, root))
	})

	t.Run("accepts descendant", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		assert.True(t, Contains(root, filepath.Join(root, "child")))
	})

	t.Run("rejects sibling", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		assert.False(t, Contains(filepath.Join(root, "left"), filepath.Join(root, "right")))
	})
}
