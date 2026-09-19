// Package build configures and runs the static-site build command.
package build

import (
	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/cli/internal/site"
	"github.com/kumbuka-me/kumbuka/pkg/domain"
	"github.com/kumbuka-me/kumbuka/pkg/logging"
)

// Config contains the resolved build command configuration.
type Config struct {
	// Site contains the effective static-site configuration.
	Site site.Config
	// LogFormat selects text or JSON command logging.
	LogFormat logging.LogFormat
}

// BindFlags registers build flags and returns a resolver for the effective configuration.
func BindFlags(flags *tinyflags.FlagSet) func() (Config, error) {
	defaults := site.DefaultConfig()

	configPath := flags.String("config", site.DefaultConfigPath, "TOML site configuration file").
		Placeholder("FILE")
	siteName := flags.String("site-name", defaults.SiteName, "Site title").
		NotEmpty().
		Placeholder("NAME")
	siteURL := flags.String("site-url", defaults.SiteURL, "Published site URL").Placeholder("URL")
	source := flags.String("source", defaults.SourceDir, "Markdown source directory").
		NotEmpty().
		Placeholder("DIR")
	output := flags.String("output", defaults.OutputDir, "Generated site directory").
		NotEmpty().
		Placeholder("DIR")
	theme := flags.String("theme", defaults.Theme, "Theme").
		NotEmpty().
		Placeholder("THEME")
	language := flags.String("language", defaults.Language, "HTML content language").
		NotEmpty().
		Placeholder("LANG")
	navigationStyle := flags.String("navigation-style", defaults.NavigationStyle, "Desktop navigation style").
		Choices(domain.NavigationStyleSidebar, domain.NavigationStyleTopbar, domain.NavigationStyleTree).
		Placeholder("STYLE")
	navigationDensity := flags.String("navigation-density", defaults.NavigationDensity, "Navigation density").
		Choices(domain.NavigationDensityComfortable, domain.NavigationDensityCompact).
		Placeholder("DENSITY")
	sidebarWidth := flags.Int("sidebar-width", defaults.SidebarWidth, "Desktop sidebar width in pixels").
		Validate(site.ValidateSidebarWidth).
		Placeholder("PIXELS")
	pluginsFile := flags.String("plugins", defaults.PluginsFile, "Static plugin dependency file").
		NotEmpty().
		Placeholder("FILE")
	robots := flags.String("robots", defaults.RobotsPolicy, "robots.txt policy").
		Choices(domain.RobotsPolicyAllow, domain.RobotsPolicyDisallow, domain.RobotsPolicyNone).
		Placeholder("POLICY")
	logFormat := flags.String("log-format", string(logging.LogFormatJSON), "Log output format").
		Choices(string(logging.LogFormatText), string(logging.LogFormatJSON)).
		Short("l").
		Placeholder("FORMAT")

	return func() (Config, error) {
		cfg, err := site.LoadConfig(*configPath.Value(), configPath.Changed())
		if err != nil {
			return Config{}, err
		}

		if siteName.Changed() {
			cfg.SiteName = *siteName.Value()
		}
		if siteURL.Changed() {
			cfg.SiteURL = *siteURL.Value()
		}
		if source.Changed() {
			cfg.SourceDir = *source.Value()
		}
		if output.Changed() {
			cfg.OutputDir = *output.Value()
		}
		if theme.Changed() {
			cfg.Theme = *theme.Value()
		}
		if language.Changed() {
			cfg.Language = *language.Value()
		}
		if navigationStyle.Changed() {
			cfg.NavigationStyle = *navigationStyle.Value()
		}
		if navigationDensity.Changed() {
			cfg.NavigationDensity = *navigationDensity.Value()
		}
		if sidebarWidth.Changed() {
			cfg.SidebarWidth = *sidebarWidth.Value()
		}
		if pluginsFile.Changed() {
			cfg.PluginsFile = *pluginsFile.Value()
		}
		if robots.Changed() {
			cfg.RobotsPolicy = *robots.Value()
		}
		if err := site.ValidateConfig(cfg); err != nil {
			return Config{}, err
		}

		return Config{
			Site:      cfg,
			LogFormat: logging.LogFormat(*logFormat.Value()),
		}, nil
	}
}
