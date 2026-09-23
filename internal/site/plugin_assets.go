package site

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/url"
	"path/filepath"

	md "github.com/kumbuka-me/kumbuka/pkg/markdown"
	"github.com/kumbuka-me/kumbuka/pkg/pluginbrowser"
)

// copyPluginAssets publishes active browser plugin assets and self-contained sandbox frames for static hosting.
func copyPluginAssets(renderer *md.Renderer, config Config, basePath string) error {
	manager := renderer.PluginManager()
	if manager == nil {
		return writeFile(filepath.Join(config.OutputDir, "plugins", "styles.css"), nil)
	}

	modules := manager.BrowserModules()
	if len(modules) == 0 {
		return writeFile(filepath.Join(config.OutputDir, "plugins", "styles.css"), []byte(pluginbrowser.PresentationStyles(manager)))
	}

	origin, err := staticPluginOrigin(config.SiteURL)
	if err != nil {
		return err
	}
	prefix := publicURLPath(basePath, "plugins")
	runtime := publicURLPath(basePath, "assets/js/plugins/frame.js")
	for _, module := range modules {
		names, err := manager.BrowserAssetNames(module.PluginID, module.Digest)
		if err != nil {
			return err
		}
		directory := filepath.Join(config.OutputDir, "plugins", module.PluginID, module.Digest)
		for _, name := range names {
			data, err := manager.BrowserAsset(module.PluginID, module.Digest, name)
			if err != nil {
				return err
			}
			if err := writeFile(filepath.Join(directory, "assets", filepath.FromSlash(name)), data); err != nil {
				return err
			}
		}
		frame, _, err := pluginbrowser.Frame(prefix, runtime, []string{origin}, module)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(directory, "frames", module.ModuleID+".html"), frame); err != nil {
			return err
		}
	}
	return writeFile(filepath.Join(config.OutputDir, "plugins", "styles.css"), []byte(pluginbrowser.PresentationStyles(manager)))
}

// staticPluginOrigin returns the absolute deployment origin required by browser-plugin frame policies.
func staticPluginOrigin(siteURL string) (string, error) {
	parsed, err := url.Parse(siteURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("site_url is required when browser plugins are enabled")
	}

	return parsed.Scheme + "://" + parsed.Host, nil
}

// pluginModulesJSON serializes the current browser module catalog for embedding in generated pages.
func pluginModulesJSON(renderer *md.Renderer, prefix string) (template.JS, error) {
	data, err := json.Marshal(pluginbrowser.Catalog(prefix, renderer.PluginManager()))
	if err != nil {
		return "", err
	}

	return template.JS(data), nil
}
