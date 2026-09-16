package build

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kumbuka-me/cli/internal/site"
	"github.com/kumbuka-me/kumbuka/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLogsOverrides(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "docs")
	output := filepath.Join(root, "site")

	require.NoError(t, os.MkdirAll(source, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "index.md"), []byte("# Home\n"), 0o644))

	config := site.DefaultConfig()
	config.SourceDir = source
	config.OutputDir = output

	var stdout bytes.Buffer
	err := Run(
		context.Background(),
		Config{Site: config, LogFormat: logging.LogFormatJSON},
		map[string]any{"site-name": "Example Docs"},
		&stdout,
	)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "CLI Overrides")
	assert.Contains(t, stdout.String(), "cli_overrides")
	assert.Contains(t, stdout.String(), "site-name")
	assert.Contains(t, stdout.String(), "Built 1 pages")
}
