package site

import (
	"encoding/json"
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
	origin, err := url.Parse(config.SiteURL)
	if err != nil {
		return err
	}
	prefix := basePath + "plugins"
	runtime := basePath + "assets/js/plugins/frame.js"
	for _, module := range manager.BrowserModules() {
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
		frame, _, err := pluginbrowser.Frame(prefix, runtime, []string{origin.Scheme + "://" + origin.Host}, module)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(directory, "frames", module.ModuleID+".html"), frame); err != nil {
			return err
		}
	}
	return writeFile(filepath.Join(config.OutputDir, "plugins", "styles.css"), []byte(pluginbrowser.PresentationStyles(manager)))
}

// pluginModulesJSON serializes the current browser module catalog for embedding in generated pages.
func pluginModulesJSON(renderer *md.Renderer, prefix string) (template.JS, error) {
	data, err := json.Marshal(pluginbrowser.Catalog(prefix, renderer.PluginManager()))
	if err != nil {
		return "", err
	}

	return template.JS(data), nil
}
