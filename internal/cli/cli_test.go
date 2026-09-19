package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShowsRootHelpWhenCommandIsMissing(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := Run(context.Background(), nil, "test", io.Discard, &stderr)

	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "build")
	assert.Contains(t, stderr.String(), "mirror")
	assert.Contains(t, stderr.String(), "plugins")
	assert.NotContains(t, stderr.String(), "serve")
}

func TestRunShowsBuildHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := Run(context.Background(), []string{"build", "--help"}, "test", &stdout, io.Discard)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "--config")
	assert.Contains(t, stdout.String(), "--site-name")
	assert.Contains(t, stdout.String(), "--plugins")
}

func TestRunListsProjectPlugins(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), ".kumbukaplugins")
	require.NoError(t, os.WriteFile(filename, []byte(`format = 1

[[plugin]]
id = "me.kumbuka.mermaid"
repository = "kumbuka-me/plugins"
tag_prefix = "mermaid/v"
asset = "mermaid"
version = "1.0.1"
`), 0o644))

	var stdout bytes.Buffer
	err := Run(context.Background(), []string{"plugins", "list", "--file", filename}, "test", &stdout, io.Discard)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "me.kumbuka.mermaid 1.0.1")
	assert.Contains(t, stdout.String(), "kumbuka-me/plugins@mermaid/v1.0.1")
}

func TestRunPrintsCommandExecutionErrors(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), ".kumbukaplugins")
	require.NoError(t, os.WriteFile(filename, []byte("not valid toml = ["), 0o644))

	var stderr bytes.Buffer
	err := Run(
		context.Background(),
		[]string{"plugins", "list", "--file", filename},
		"test",
		io.Discard,
		&stderr,
	)

	require.Error(t, err)
	assert.Contains(t, stderr.String(), "parse "+filename)
}
