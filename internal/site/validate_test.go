package site

import (
	"testing"

	"github.com/kumbuka-me/kumbuka/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFileValuesRejectEmptySettings(t *testing.T) {
	t.Parallel()

	t.Run("site name", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.SiteName = ""

		assert.ErrorContains(t, validateConfigValues(config), "site_name must not be empty")
	})

	t.Run("source directory", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.SourceDir = ""

		assert.ErrorContains(t, validateConfigValues(config), "source_dir must not be empty")
	})

	t.Run("output directory", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.OutputDir = ""

		assert.ErrorContains(t, validateConfigValues(config), "output_dir must not be empty")
	})

	t.Run("theme", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.Theme = ""

		assert.ErrorContains(t, validateConfigValues(config), "theme must not be empty")
	})

	t.Run("language", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.Language = ""

		assert.ErrorContains(t, validateConfigValues(config), "language must not be empty")
	})
}

func TestValidateConfigAppliesCompleteValidation(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid scalar values", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.NavigationStyle = "columns"

		assert.ErrorContains(t, ValidateConfig(config), "navigation_style")
	})

	t.Run("rejects invalid external links", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.ExternalLinks = []domain.ExternalLink{{Label: "Repository", URL: "javascript:alert(1)"}}

		assert.ErrorContains(t, ValidateConfig(config), "HTTP or HTTPS")
	})

	t.Run("rejects invalid hover effects after normalization", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.ExternalLinks = []domain.ExternalLink{{Label: "Repository", URL: "https://example.test", HoverEffect: " bounce "}}

		assert.ErrorContains(t, ValidateConfig(config), "hover_effect")
	})
}

func TestValidateConfigDoesNotMutateExternalLinks(t *testing.T) {
	t.Parallel()

	config := defaultConfig()
	config.ExternalLinks = []domain.ExternalLink{{
		Label:       " Repository ",
		URL:         " https://example.test ",
		HoverEffect: " lift ",
	}}

	require.NoError(t, ValidateConfig(config))
	assert.Equal(t, " Repository ", config.ExternalLinks[0].Label)
	assert.Equal(t, " https://example.test ", config.ExternalLinks[0].URL)
	assert.Equal(t, " lift ", config.ExternalLinks[0].HoverEffect)
}
