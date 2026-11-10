package serve

import (
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeFlags(t *testing.T) {
	t.Parallel()

	flags := tinyflags.NewFlagSet("kumbuka-cli serve", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)
	require.NoError(t, flags.Parse([]string{"--listen-address", "127.0.0.1:9000", "--no-build", "--no-watch"}))
	config, err := resolve()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9000", config.ListenAddress)
	assert.True(t, config.NoBuild)
	assert.True(t, config.NoWatch)
}
