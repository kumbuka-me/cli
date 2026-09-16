package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/cli/internal/site"
	"github.com/kumbuka-me/kumbuka/pkg/domain"
	"github.com/kumbuka-me/kumbuka/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagsOverrideConfigurationFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "site.toml")

	require.NoError(t, os.WriteFile(configPath, []byte(`
site_name = "From file"
site_url = "https://example.com/docs/"
source_dir = "docs"
output_dir = "site"
theme = "Light"
language = "en"
navigation_style = "tree"
navigation_density = "compact"
sidebar_width = 360
`), 0o600))

	flags := tinyflags.NewFlagSet("kumbuka build", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--config", configPath,
		"--site-name", "From CLI",
		"--navigation-style", "topbar",
		"--navigation-density", "comfortable",
		"--sidebar-width", "320",
		"--robots=disallow",
		"--log-format=text",
	}))

	cfg, err := resolve()

	require.NoError(t, err)
	assert.Equal(t, "From CLI", cfg.Site.SiteName)
	assert.Equal(t, "https://example.com/docs/", cfg.Site.SiteURL)
	assert.Equal(t, domain.NavigationStyleTopbar, cfg.Site.NavigationStyle)
	assert.Equal(t, domain.NavigationDensityComfortable, cfg.Site.NavigationDensity)
	assert.Equal(t, 320, cfg.Site.SidebarWidth)
	assert.Equal(t, domain.RobotsPolicyDisallow, cfg.Site.RobotsPolicy)
	assert.Equal(t, logging.LogFormatText, cfg.LogFormat)
}

func TestFlagsAllowDefaults(t *testing.T) {
	t.Parallel()

	flags := tinyflags.NewFlagSet("kumbuka build", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse(nil))

	config, err := resolve()

	require.NoError(t, err)
	assert.Equal(t, site.DefaultConfig(), config.Site)
	assert.Equal(t, logging.LogFormatJSON, config.LogFormat)
}

func TestFlagsValidateValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "blank site name", args: []string{"--site-name", ""}, want: "flag --site-name must not be empty"},
		{name: "blank source", args: []string{"--source", ""}, want: "flag --source must not be empty"},
		{name: "blank output", args: []string{"--output", ""}, want: "flag --output must not be empty"},
		{name: "blank theme", args: []string{"--theme", ""}, want: "flag --theme must not be empty"},
		{name: "blank language", args: []string{"--language", ""}, want: "flag --language must not be empty"},
		{name: "sidebar below range", args: []string{"--sidebar-width", "219"}, want: "sidebar_width must be between 220 and 420 pixels"},
		{name: "sidebar above range", args: []string{"--sidebar-width", "421"}, want: "sidebar_width must be between 220 and 420 pixels"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			flags := tinyflags.NewFlagSet("kumbuka build", tinyflags.ContinueOnError)
			_ = BindFlags(flags)

			err := flags.Parse(test.args)

			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestFlagsValidateOverrides(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	flags := tinyflags.NewFlagSet("kumbuka build", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--source", filepath.Join(root, "docs"),
		"--output", filepath.Join(root, "docs", "site"),
	}))

	_, err := resolve()

	assert.ErrorContains(t, err, "source_dir and output_dir must be separate directories")
}
