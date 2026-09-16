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
	Logo              string                `toml:"logo"`
	Favicon           string                `toml:"favicon"`
	FaviconICO        string                `toml:"favicon_ico"`
	AssetsDir         string                `toml:"assets_dir"`
	SiteName          string                `toml:"site_name"`
	SiteURL           string                `toml:"site_url"`
	SourceDir         string                `toml:"source_dir"`
	OutputDir         string                `toml:"output_dir"`
	Theme             string                `toml:"theme"`
	Language          string                `toml:"language"`
	NavigationStyle   string                `toml:"navigation_style"`
	NavigationDensity string                `toml:"navigation_density"`
	SidebarWidth      int                   `toml:"sidebar_width"`
	RobotsPolicy      string                `toml:"robots"`
	ExternalLinks     []domain.ExternalLink `toml:"external_links"`
	PluginsFile       string                `toml:"-"`
}

// DefaultConfig returns generic zero-infrastructure static site defaults.
func DefaultConfig() Config {
	return defaultConfig()
}

// defaultConfig returns generic zero-infrastructure static site defaults.
func defaultConfig() Config {
	preferences := domain.DefaultUserPreferences()

	return Config{
		SiteName:          "Documentation",
		SourceDir:         "docs",
		OutputDir:         "site",
		Theme:             themes.DefaultTheme,
		Language:          "en",
		NavigationStyle:   preferences.NavigationStyle,
		NavigationDensity: preferences.NavigationDensity,
		SidebarWidth:      preferences.SidebarWidth,
		RobotsPolicy:      domain.RobotsPolicyAllow,
		PluginsFile:       pluginproject.DefaultFile,
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
	if err := validateConfigFile(config); err != nil {
		return Config{}, err
	}

	return config, nil
}

// ValidateConfig validates relationships after command-line overrides have been applied.
func ValidateConfig(config Config) error {
	return validateResolvedConfig(config)
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
