package site

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kumbuka-me/cli/internal/pluginproject"
	"github.com/kumbuka-me/kumbuka/pkg/domain"
	"github.com/kumbuka-me/kumbuka/pkg/themes"
	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigPath = "kumbuka-site.toml"

// Config contains filesystem-backed static site build settings.
type Config struct {
	// Logo is the optional site logo file.
	Logo string `toml:"logo"`
	// Favicon is the optional modern favicon file.
	Favicon string `toml:"favicon"`
	// FaviconICO is the optional legacy ICO favicon file.
	FaviconICO string `toml:"favicon_ico"`
	// AssetsDir is an optional directory copied below the generated assets path.
	AssetsDir string `toml:"assets_dir"`
	// SiteName is the title displayed by generated pages.
	SiteName string `toml:"site_name"`
	// Footer is optional plain text displayed below generated pages.
	Footer string `toml:"footer"`
	// SiteURL is the published base URL used to derive generated paths and sitemap URLs.
	SiteURL string `toml:"site_url"`
	// SourceDir contains Markdown pages and source assets.
	SourceDir string `toml:"source_dir"`
	// OutputDir receives the generated static site.
	OutputDir string `toml:"output_dir"`
	// Theme selects the initial Kumbuka theme.
	Theme string `toml:"theme"`
	// Language becomes the generated HTML content language.
	Language string `toml:"language"`
	// NavigationStyle selects the desktop navigation layout.
	NavigationStyle domain.NavigationStyle `toml:"navigation_style"`
	// NavigationDensity selects the navigation spacing preset.
	NavigationDensity domain.NavigationDensity `toml:"navigation_density"`
	// SidebarWidth is the desktop sidebar width in pixels.
	SidebarWidth int `toml:"sidebar_width"`
	// ExpandContentWhenHidden lets static pages reclaim hidden navigation and page-contents space.
	ExpandContentWhenHidden bool `toml:"expand_content_when_hidden"`
	// RobotsPolicy controls generated robots.txt content.
	RobotsPolicy domain.RobotsPolicy `toml:"robots"`
	// ExternalLinks contains configured top-bar links.
	ExternalLinks []domain.ExternalLink `toml:"external_links"`
	// PluginsFile is the project dependency file selected outside TOML configuration.
	PluginsFile string `toml:"-"`
}

// DefaultConfig returns generic zero-infrastructure static site defaults.
func DefaultConfig() Config {
	return defaultConfig()
}

// defaultConfig returns generic zero-infrastructure static site defaults.
func defaultConfig() Config {
	preferences := domain.DefaultUserPreferences()

	return Config{
		SiteName:                "Documentation",
		SourceDir:               "docs",
		OutputDir:               "site",
		Theme:                   themes.DefaultTheme,
		Language:                "en",
		NavigationStyle:         preferences.NavigationStyle,
		NavigationDensity:       preferences.NavigationDensity,
		SidebarWidth:            preferences.SidebarWidth,
		ExpandContentWhenHidden: true,
		RobotsPolicy:            domain.RobotsPolicyAllow,
		PluginsFile:             pluginproject.DefaultFile,
	}
}

// LoadConfig reads an optional TOML site configuration and resolves config-relative asset paths.
func LoadConfig(filename string, required bool) (Config, error) {
	return loadConfig(filename, required)
}

// loadConfig reads an optional TOML site configuration and resolves config-relative asset paths.
func loadConfig(filename string, required bool) (Config, error) {
	config := defaultConfig()
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) && !required {
		return config, nil
	}
	if err != nil {
		return Config{}, err
	}
	defer file.Close() // nolint:errcheck

	if err := toml.NewDecoder(file).DisallowUnknownFields().Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	config.resolveAssetPaths(filepath.Dir(filename))
	config = normalizeConfig(config)
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}

	return config, nil
}

// ValidateConfig validates the complete effective static-site configuration without mutating it.
func ValidateConfig(config Config) error {
	return validateConfig(normalizeConfig(config))
}

// ValidateSidebarWidth checks the supported static navigation width range.
func ValidateSidebarWidth(width int) error {
	return validateSidebarWidth(width)
}

// resolveAssetPaths resolves user-supplied branding and asset paths relative to the config file.
func (c *Config) resolveAssetPaths(configDir string) {
	resolveRelativePath(configDir, &c.Logo)
	resolveRelativePath(configDir, &c.Favicon)
	resolveRelativePath(configDir, &c.FaviconICO)
	resolveRelativePath(configDir, &c.AssetsDir)
}

// resolveRelativePath resolves one non-empty relative path against the supplied directory.
func resolveRelativePath(baseDir string, filename *string) {
	if *filename == "" || filepath.IsAbs(*filename) {
		return
	}
	*filename = filepath.Clean(filepath.Join(baseDir, *filename))
}
