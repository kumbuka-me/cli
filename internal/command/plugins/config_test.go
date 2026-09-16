package plugins

import (
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/cli/internal/pluginproject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFlagsUseProjectFileDefault(t *testing.T) {
	t.Parallel()

	flags := tinyflags.NewFlagSet("kumbuka plugins list", tinyflags.ContinueOnError)
	resolve := BindListFlags(flags)

	require.NoError(t, flags.Parse(nil))
	assert.Equal(t, pluginproject.DefaultFile, resolve().File)
}

func TestAddFlags(t *testing.T) {
	t.Parallel()

	flags := tinyflags.NewFlagSet("kumbuka plugins add", tinyflags.ContinueOnError)
	resolve := BindAddFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--id", "com.example.chart",
		"--repository", "example/chart",
		"--plugin-version", "2.3.0",
		"--tag-prefix", "chart/v",
		"--asset", "chart",
	}))

	config := resolve()
	assert.Equal(t, "com.example.chart", config.ID)
	assert.Equal(t, "example/chart", config.Repository)
	assert.Equal(t, "2.3.0", config.Version)
	assert.Equal(t, "chart/v", config.TagPrefix)
	assert.Equal(t, "chart", config.Asset)
}
