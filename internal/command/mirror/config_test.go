package mirror

import (
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/kumbuka/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlags(t *testing.T) {
	t.Parallel()

	flags := tinyflags.NewFlagSet("kumbuka mirror", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--database-url", "postgres://user:pass@localhost/kumbuka",
		"--output", "snapshot",
		"--log-format=json",
	}))

	config := resolve()
	assert.Equal(t, "postgres://user:pass@localhost/kumbuka", config.DatabaseURL)
	assert.Equal(t, "snapshot", config.OutputDir)
	assert.Equal(t, logging.LogFormatJSON, config.LogFormat)
}
