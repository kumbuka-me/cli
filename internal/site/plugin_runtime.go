package site

import (
	"context"
	"fmt"

	"github.com/kumbuka-me/cli/internal/pluginproject"
	md "github.com/kumbuka-me/kumbuka/pkg/markdown"
	"github.com/kumbuka-me/sdk/pluginpackage"
)

// projectRenderer resolves and selects only project plugins required by the discovered Markdown.
func projectRenderer(ctx context.Context, filename string, pages []sourcePage) (*md.Renderer, error) {
	file, err := pluginproject.Load(filename)
	if err != nil {
		return nil, err
	}
	if len(file.Plugins) == 0 {
		return md.NewWithPluginPackages(ctx, nil, nil)
	}

	resolver, err := pluginproject.NewResolver(nil)
	if err != nil {
		return nil, err
	}
	resolved, err := resolver.Resolve(ctx, file.Plugins)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", filename, err)
	}

	required := md.RequiredPluginIDs(pageMarkdownSources(pages), resolvedManifests(resolved))
	selected, err := pluginproject.Select(resolved, required)
	if err != nil {
		return nil, err
	}

	archives, ids := pluginPackageInputs(selected)
	return md.NewWithPluginPackages(ctx, archives, ids)
}

// resolvedManifests extracts validated manifests from resolved project plugins.
func resolvedManifests(resolved []pluginproject.Resolved) []pluginpackage.Manifest {
	manifests := make([]pluginpackage.Manifest, 0, len(resolved))
	for _, item := range resolved {
		manifests = append(manifests, item.Manifest)
	}
	return manifests
}

// pageMarkdownSources extracts Markdown source text from discovered static pages.
func pageMarkdownSources(pages []sourcePage) []string {
	sources := make([]string, 0, len(pages))
	for _, page := range pages {
		sources = append(sources, page.Markdown)
	}
	return sources
}

// pluginPackageInputs extracts package archives and matching manifest IDs in the same order.
func pluginPackageInputs(selected []pluginproject.Resolved) ([][]byte, []string) {
	archives := make([][]byte, 0, len(selected))
	ids := make([]string, 0, len(selected))
	for _, item := range selected {
		archives = append(archives, item.Archive)
		ids = append(ids, item.Manifest.ID)
	}
	return archives, ids
}
