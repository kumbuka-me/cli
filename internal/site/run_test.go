package site

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "docs")
	output := filepath.Join(root, "site")

	require.NoError(t, os.MkdirAll(source, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "index.md"), []byte("# Home\n"), 0o644))

	config := defaultConfig()
	config.SourceDir = source
	config.OutputDir = output

	result, err := Build(context.Background(), config)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Pages)
	assert.Equal(t, output, result.OutputDir)
}
