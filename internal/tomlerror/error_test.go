package tomlerror

import (
	"errors"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplainStrictUnknownFields(t *testing.T) {
	t.Parallel()

	var config struct {
		SiteName string `toml:"site_name"`
		Links    []struct {
			Label string `toml:"label"`
		} `toml:"links"`
	}

	document := `site_name = "Documentation"
mermaid = true
legacy_setting = false

[[links]]
label = "Repository"
unsupported = "value"
`
	err := toml.NewDecoder(strings.NewReader(document)).DisallowUnknownFields().Decode(&config)
	require.Error(t, err)

	explained := Explain(err)
	assert.ErrorContains(t, explained, "unknown TOML fields:")
	assert.ErrorContains(t, explained, "line 2, column 1: \"mermaid\"")
	assert.ErrorContains(t, explained, "line 3, column 1: \"legacy_setting\"")
	assert.ErrorContains(t, explained, "line 7, column 1: \"links.unsupported\"")
	assert.NotContains(t, explained.Error(), "fields in the document are missing")

	var original *toml.StrictMissingError
	assert.ErrorAs(t, explained, &original)
}

func TestExplainPreservesNonStrictErrors(t *testing.T) {
	t.Parallel()

	var config struct {
		SiteName string `toml:"site_name"`
	}

	err := toml.NewDecoder(strings.NewReader("site_name = [\n")).DisallowUnknownFields().Decode(&config)
	require.Error(t, err)
	assert.Error(t, err)
	assert.ErrorIs(t, Explain(err), err)
	assert.Equal(t, err, Explain(err))

	plain := errors.New("unrelated failure")
	assert.ErrorIs(t, Explain(plain), plain)
}

func TestExplainSingleUnknownField(t *testing.T) {
	t.Parallel()

	var config struct {
		SiteName string `toml:"site_name"`
	}

	err := toml.NewDecoder(strings.NewReader("mermaid = true\n")).DisallowUnknownFields().Decode(&config)
	require.Error(t, err)

	explained := Explain(err)
	assert.Equal(t, "unknown TOML field:\n  line 1, column 1: \"mermaid\"", explained.Error())

	var original *toml.StrictMissingError
	assert.ErrorAs(t, explained, &original)
}
