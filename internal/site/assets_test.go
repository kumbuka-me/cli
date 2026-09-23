package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishConfiguredFile(t *testing.T) {
	t.Parallel()

	t.Run("keeps source directory path", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		contentDir := filepath.Join(root, "content")
		filename := filepath.Join(contentDir, "images", "favicon.svg")
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
		require.NoError(t, os.WriteFile(filename, []byte("favicon"), 0o644))

		config := defaultConfig()
		config.SourceDir = contentDir
		config.OutputDir = filepath.Join(root, "site")

		publicPath, err := publishConfiguredFile(config, filename)

		require.NoError(t, err)
		assert.Equal(t, "images/favicon.svg", publicPath)
	})

	t.Run("keeps assets directory path", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		assetsDir := filepath.Join(root, "branding")
		filename := filepath.Join(assetsDir, "company", "logo.svg")
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
		require.NoError(t, os.WriteFile(filename, []byte("logo"), 0o644))

		config := defaultConfig()
		config.AssetsDir = assetsDir
		config.OutputDir = filepath.Join(root, "site")

		publicPath, err := publishConfiguredFile(config, filename)

		require.NoError(t, err)
		assert.Equal(t, "assets/company/logo.svg", publicPath)
	})

	t.Run("publishes external file by its own name", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		filename := filepath.Join(root, "branding", "company.svg")
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
		require.NoError(t, os.WriteFile(filename, []byte("logo"), 0o644))

		config := defaultConfig()
		config.SourceDir = filepath.Join(root, "content")
		config.OutputDir = filepath.Join(root, "site")
		require.NoError(t, os.MkdirAll(config.SourceDir, 0o755))

		publicPath, err := publishConfiguredFile(config, filename)

		require.NoError(t, err)
		assert.Equal(t, "assets/company.svg", publicPath)
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "assets", "company.svg"))
		require.NoError(t, err)
		assert.Equal(t, "logo", string(data))
	})
}

func TestCopySourceAssetsRejectsSymbolicLinks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "assets")
	output := filepath.Join(root, "site")
	require.NoError(t, os.MkdirAll(source, 0o755))
	external := filepath.Join(root, "outside.txt")
	require.NoError(t, os.WriteFile(external, []byte("outside"), 0o644))
	if err := os.Symlink(external, filepath.Join(source, "linked.txt")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	err := copySourceAssets(source, output)

	require.ErrorContains(t, err, "symbolic links are not supported")
	_, statErr := os.Stat(filepath.Join(output, "linked.txt"))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestStaticPluginOrigin(t *testing.T) {
	t.Parallel()

	t.Run("returns configured deployment origin", func(t *testing.T) {
		t.Parallel()

		origin, err := staticPluginOrigin("https://docs.example.com/kumbuka/")

		require.NoError(t, err)
		assert.Equal(t, "https://docs.example.com", origin)
	})

	t.Run("requires site URL for browser plugins", func(t *testing.T) {
		t.Parallel()

		_, err := staticPluginOrigin("")

		require.ErrorContains(t, err, "site_url is required")
	})
}
