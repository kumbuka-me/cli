package pluginproject

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectFileRoundTrip(t *testing.T) {
	filename := filepath.Join(t.TempDir(), DefaultFile)
	require.NoError(t, Add(filename, Dependency{
		ID:         "me.kumbuka.mermaid",
		Repository: "kumbuka-me/plugins",
		TagPrefix:  "mermaid/v",
		Asset:      "mermaid",
		Version:    "1.0.1",
	}))
	require.NoError(t, Add(filename, Dependency{
		ID:         "com.example.chart",
		Repository: "example/chart",
		Version:    "v2.3.0",
	}))

	file, err := Load(filename)
	require.NoError(t, err)
	require.Len(t, file.Plugins, 2)
	assert.Equal(t, "com.example.chart", file.Plugins[0].ID)
	assert.Equal(t, "v", file.Plugins[0].TagPrefix)
	assert.Equal(t, "chart", file.Plugins[0].Asset)
	assert.Equal(t, "2.3.0", file.Plugins[0].Version)

	require.NoError(t, Remove(filename, "me.kumbuka.mermaid"))
	file, err = Load(filename)
	require.NoError(t, err)
	require.Len(t, file.Plugins, 1)
	assert.Equal(t, "com.example.chart", file.Plugins[0].ID)

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Contains(t, string(data), "format = 1")
	assert.Contains(t, string(data), "[[plugin]]")
}

func TestProjectFileRejectsDuplicates(t *testing.T) {
	filename := filepath.Join(t.TempDir(), DefaultFile)
	require.NoError(t, os.WriteFile(filename, []byte(`format = 1

[[plugin]]
id = "io.example"
repository = "example/plugins"
tag_prefix = "example/v"
asset = "example"
version = "1.0.0"

[[plugin]]
id = "io.example"
repository = "example/plugins"
tag_prefix = "example/v"
asset = "example"
version = "1.0.1"
`), 0o644))

	_, err := Load(filename)
	require.ErrorContains(t, err, "duplicate plugin io.example")
}

func TestSaveDoesNotMutateCaller(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), DefaultFile)
	file := File{Plugins: []Dependency{
		{ID: "z.example", Repository: " example/z.git ", Version: "v1.0.0"},
		{ID: "a.example", Repository: "example/a", Version: "2.0.0"},
	}}
	before := append([]Dependency(nil), file.Plugins...)

	require.NoError(t, Save(filename, file))

	assert.Equal(t, before, file.Plugins)
	loaded, err := Load(filename)
	require.NoError(t, err)
	assert.Equal(t, "a.example", loaded.Plugins[0].ID)
	assert.Equal(t, "z.example", loaded.Plugins[1].ID)
	assert.Equal(t, "example/z", loaded.Plugins[1].Repository)
	assert.Equal(t, "1.0.0", loaded.Plugins[1].Version)
}
