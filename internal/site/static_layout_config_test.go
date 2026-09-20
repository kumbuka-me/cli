package site

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/kumbuka-me/kumbuka/pkg/icons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandContentWhenHiddenConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("enabled by default", func(t *testing.T) {
		t.Parallel()

		assert.True(t, defaultConfig().ExpandContentWhenHidden)
	})

	t.Run("can be disabled in site config", func(t *testing.T) {
		t.Parallel()

		filename := filepath.Join(t.TempDir(), "site.toml")
		require.NoError(t, os.WriteFile(
			filename,
			[]byte("expand_content_when_hidden = false\n"),
			0o600,
		))

		config, err := loadConfig(filename, true)

		require.NoError(t, err)
		assert.False(t, config.ExpandContentWhenHidden)
	})
}

func TestExpandContentWhenHiddenReachesStaticLayout(t *testing.T) {
	t.Parallel()

	config := defaultConfig()
	common := commonViewData(buildPlan{config: config, basePath: "/"}, brandingData{})
	assert.True(t, common.ExpandContentWhenHidden)

	templates, err := parseTemplates("/", icons.Builtin())
	require.NoError(t, err)

	common.Language = "en"
	common.Title = "Home"
	common.SiteName = "Documentation"
	common.ActiveTheme = "system"

	var output bytes.Buffer
	require.NoError(t, templates.page.ExecuteTemplate(&output, "layout", common))
	assert.Contains(t, output.String(), `data-expand-content-when-hidden="true"`)
	assert.Contains(t, output.String(), `href="/assets/css/static-layout.css"`)

	stylesheet, err := fs.ReadFile(staticAssets, "css/static-layout.css")
	require.NoError(t, err)
	assert.Contains(t, string(stylesheet), `data-expand-content-when-hidden="true"`)
	assert.Contains(t, staticBrowserAssets, "css/static-layout.css")
}

func TestFooterReachesStaticLayout(t *testing.T) {
	t.Parallel()

	t.Run("renders configured plain text", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.Footer = `Built with <Kumbuka>`
		common := commonViewData(buildPlan{config: config, basePath: "/"}, brandingData{})

		templates, err := parseTemplates("/", icons.Builtin())
		require.NoError(t, err)

		common.Language = "en"
		common.Title = "Home"
		common.SiteName = "Documentation"
		common.ActiveTheme = "system"

		var output bytes.Buffer
		require.NoError(t, templates.page.ExecuteTemplate(&output, "layout", common))
		assert.Contains(t, output.String(), "<footer>Built with &lt;Kumbuka&gt;</footer>")
	})

	t.Run("omits footer when not configured", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		common := commonViewData(buildPlan{config: config, basePath: "/"}, brandingData{})

		templates, err := parseTemplates("/", icons.Builtin())
		require.NoError(t, err)

		common.Language = "en"
		common.Title = "Home"
		common.SiteName = "Documentation"
		common.ActiveTheme = "system"

		var output bytes.Buffer
		require.NoError(t, templates.page.ExecuteTemplate(&output, "layout", common))
		assert.NotContains(t, output.String(), "<footer>")
	})
}
