package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kumbuka-me/kumbuka/pkg/plugin"
	"github.com/kumbuka-me/sdk/staticexport"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// staticModule binds an advertised exporter to one exact-version browser module.
type staticModule struct {
	// module identifies the installed browser module and its asset location.
	module plugin.BrowserContribution
	// mediaType is the output format advertised by the plugin.
	mediaType string
}

// staticAsset describes a validated, published, content-addressed artifact pair.
type staticAsset struct {
	// key is the content-addressed output filename stem.
	key string
	// alt describes the exported image to assistive technology.
	alt string
	// alignment is the bounded preferred presentation alignment.
	alignment string
}

// staticExporter converts browser plugin blocks through optional, sandboxed plugin exports.
type staticExporter struct {
	// modules contains only validated, opted-in browser modules.
	modules map[string]staticModule
	// cached avoids repeated browser rendering of identical blocks within a build.
	cached map[string]*staticAsset
	// outputDir is the staged static site root.
	outputDir string
	// basePath is the public URL prefix of the generated site.
	basePath string
	// runner lazily starts the isolated headless browser when a supported block appears.
	runner staticExportRuntime
}

// newStaticExporter discovers optional export declarations without changing the plugin manifest contract.
func newStaticExporter(ctx context.Context, manager *plugin.Manager, outputDir, basePath string) (*staticExporter, error) {
	exporter := &staticExporter{
		modules:   make(map[string]staticModule),
		cached:    make(map[string]*staticAsset),
		outputDir: outputDir,
		basePath:  basePath,
	}
	if manager == nil {
		return exporter, nil
	}

	for _, module := range manager.BrowserModules() {
		declaration, err := manager.BrowserAsset(ctx, module.PluginID, module.Digest, staticexport.DeclarationAsset)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read static exports from plugin %s: %w", module.PluginID, err)
		}
		parsed, err := staticexport.ParseDeclaration(declaration)
		if err != nil {
			return nil, fmt.Errorf("plugin %s: %w", module.PluginID, err)
		}
		for _, entry := range parsed.Modules {
			if entry.ID == module.ModuleID {
				exporter.modules[staticModuleKey(module.PluginID, module.ModuleID)] = staticModule{
					module:    module,
					mediaType: entry.MediaType,
				}
			}
		}
	}
	return exporter, nil
}

// staticModuleKey identifies one browser module without depending on its content format.
func staticModuleKey(pluginID, moduleID string) string { return pluginID + "\x00" + moduleID }

// materialize replaces exportable browser-module blocks with passive static assets.
func (e *staticExporter) materialize(ctx context.Context, markup, sourcePath string) (string, error) {
	if len(e.modules) == 0 {
		return markup, nil
	}

	fragment := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(markup), fragment)
	if err != nil {
		return "", fmt.Errorf("parse static page HTML: %w", err)
	}
	for _, node := range nodes {
		fragment.AppendChild(node)
	}
	changed, err := e.materializeChildren(ctx, fragment, sourcePath)
	if err != nil {
		return "", err
	}
	if !changed {
		return markup, nil
	}

	var result strings.Builder
	for node := fragment.FirstChild; node != nil; node = node.NextSibling {
		if err := xhtml.Render(&result, node); err != nil {
			return "", fmt.Errorf("serialize static page HTML: %w", err)
		}
	}
	return result.String(), nil
}

// materializeChildren recursively replaces eligible browser wrappers, preserving other modules unchanged.
func (e *staticExporter) materializeChildren(ctx context.Context, parent *xhtml.Node, sourcePath string) (bool, error) {
	changed := false
	for node := parent.FirstChild; node != nil; {
		next := node.NextSibling
		module, found := e.matchModule(node)
		if found {
			asset, err := e.export(ctx, node, module)
			if err != nil {
				return false, fmt.Errorf("%s: static export %s/%s: %w", sourcePath, module.module.PluginID, module.module.ModuleID, err)
			}
			if asset != nil {
				parent.InsertBefore(e.imageNode(*asset), node)
				parent.RemoveChild(node)
				changed = true
			}
		} else {
			converted, err := e.materializeChildren(ctx, node, sourcePath)
			if err != nil {
				return false, err
			}
			changed = changed || converted
		}
		node = next
	}
	return changed, nil
}

// matchModule resolves an ordinary browser-module wrapper through its generic host markers.
func (e *staticExporter) matchModule(node *xhtml.Node) (staticModule, bool) {
	if node.Type != xhtml.ElementNode || node.Data != "div" {
		return staticModule{}, false
	}
	pluginID, moduleID := "", ""
	for _, attribute := range node.Attr {
		switch attribute.Key {
		case "data-kumbuka-plugin":
			pluginID = attribute.Val
		case "data-kumbuka-module":
			moduleID = attribute.Val
		}
	}
	module, found := e.modules[staticModuleKey(pluginID, moduleID)]
	return module, found
}

// export invokes the exact module and validates both theme variants before publication.
func (e *staticExporter) export(ctx context.Context, node *xhtml.Node, module staticModule) (*staticAsset, error) {
	var content strings.Builder
	if err := xhtml.Render(&content, node); err != nil {
		return nil, fmt.Errorf("serialize browser plugin block: %w", err)
	}
	markup := content.String()
	key := staticAssetKey(module.module, markup)
	if cached, exists := e.cached[key]; exists {
		return cached, nil
	}
	if e.runner == nil {
		e.runner = &browserExportRunner{}
	}

	assetDir := filepath.Join(e.outputDir, "plugins", module.module.PluginID, module.module.Digest, "assets")
	variants := make(map[string]staticexport.Result, 2)
	for _, theme := range []string{"light", "dark"} {
		result, err := e.runner.Render(ctx, assetDir, module.module.JavaScript, markup, theme)
		if err != nil {
			return nil, err
		}
		if result == nil {
			if theme == "light" {
				e.cached[key] = nil
				return nil, nil
			}
			return nil, errors.New("static exporter omitted the dark variant")
		}
		if err := staticexport.ValidateResult(*result, module.mediaType); err != nil {
			return nil, fmt.Errorf("%s variant: %w", theme, err)
		}
		variants[theme] = *result
	}

	light := variants["light"]
	dark := variants["dark"]
	if light.Alignment != dark.Alignment || light.Alt != dark.Alt {
		return nil, errors.New("static export presentation differs between themes")
	}
	asset := &staticAsset{key: key, alt: light.Alt, alignment: light.Alignment}
	if err := e.writeVariants(*asset, variants); err != nil {
		return nil, err
	}
	e.cached[key] = asset
	return asset, nil
}

// staticAssetKey covers the plugin package digest, browser module, and full rendered input block.
func staticAssetKey(module plugin.BrowserContribution, markup string) string {
	hash := sha256.Sum256([]byte(module.PluginID + "\x00" + module.Digest + "\x00" + module.ModuleID + "\x00" + markup))
	return hex.EncodeToString(hash[:])
}

// writeVariants publishes validated SVGs beneath the static site's asset directory.
func (e *staticExporter) writeVariants(asset staticAsset, variants map[string]staticexport.Result) error {
	directory := filepath.Join(e.outputDir, "assets", "generated")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	for _, theme := range []string{"light", "dark"} {
		filename := filepath.Join(directory, asset.key+"-"+theme+".svg")
		if err := os.WriteFile(filename, []byte(variants[theme].Content), 0o644); err != nil {
			return fmt.Errorf("write generated asset: %w", err)
		}
	}
	return nil
}

// imageNode creates passive image elements selected by the site's existing light/dark stylesheet.
func (e *staticExporter) imageNode(asset staticAsset) *xhtml.Node {
	alignment := asset.alignment
	if alignment == "" {
		alignment = "center"
	}
	block := &xhtml.Node{
		Type:     xhtml.ElementNode,
		DataAtom: atom.Div,
		Data:     "div",
		Attr: []xhtml.Attribute{
			{Key: "class", Val: "kumbuka-static-illustration"},
			{Key: "style", Val: "text-align: " + alignment + ";"},
		},
	}
	for _, theme := range []string{"light", "dark"} {
		image := &xhtml.Node{
			Type:     xhtml.ElementNode,
			DataAtom: atom.Img,
			Data:     "img",
			Attr: []xhtml.Attribute{
				{Key: "class", Val: "kumbuka-static-illustration__" + theme},
				{Key: "src", Val: publicURLPath(e.basePath, "assets/generated/"+asset.key+"-"+theme+".svg")},
				{Key: "alt", Val: asset.alt},
			},
		}
		block.AppendChild(image)
	}
	return block
}

// Close stops the optional headless browser process after the static site build completes.
func (e *staticExporter) Close() error {
	if e.runner == nil {
		return nil
	}
	return e.runner.Close()
}
