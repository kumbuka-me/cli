package plugins

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/kumbuka-me/cli/internal/pluginproject"
	"github.com/stretchr/testify/require"
)

// failingWriter returns a deterministic write failure.
type failingWriter struct{}

// Write rejects every write.
func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunListReturnsOutputWriteError(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), pluginproject.DefaultFile)
	require.NoError(t, pluginproject.Save(filename, pluginproject.File{Plugins: []pluginproject.Dependency{{
		ID:         "com.example.chart",
		Repository: "example/chart",
		Version:    "2.3.0",
	}}}))

	err := RunList(ListConfig{File: filename}, failingWriter{})

	require.ErrorContains(t, err, "write plugin list")
}
