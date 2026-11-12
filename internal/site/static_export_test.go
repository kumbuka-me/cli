package site

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kumbuka-me/kumbuka/pkg/plugin"
	"github.com/kumbuka-me/sdk/staticexport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testStaticSVG = `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg>`

// staticExportStub records browser invocations and produces deterministic passive SVGs.
type staticExportStub struct {
	// calls counts the requested theme variants.
	calls int
	// fail creates an invalid SVG to exercise artifact validation.
	fail bool
}

// Render simulates a browser module opting out of interactive diagrams.
func (s *staticExportStub) Render(_ context.Context, _, _, markup, _ string) (*staticexport.Result, error) {
	s.calls++
	if strings.Contains(markup, "kumbuka-interactive: true") {
		return nil, nil
	}
	content := testStaticSVG
	if s.fail {
		content = `<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`
	}
	return &staticexport.Result{Content: content, Alt: "Test", Alignment: "left"}, nil
}

// Close is a no-op for the deterministic browser stub.
func (s *staticExportStub) Close() error { return nil }

// newStaticExporterStub constructs one opt-in module without loading a plugin package.
func newStaticExporterStub(t *testing.T, runtime staticExportRuntime) *staticExporter {
	t.Helper()
	module := plugin.BrowserContribution{
		PluginID:   "com.example.diagram",
		ModuleID:   "graph",
		Digest:     "package-digest",
		JavaScript: "plugin.js",
	}
	return &staticExporter{
		modules:   map[string]staticModule{staticModuleKey(module.PluginID, module.ModuleID): {module: module, mediaType: staticexport.SVGMediaType}},
		cached:    make(map[string]*staticAsset),
		outputDir: t.TempDir(),
		basePath:  "/docs/",
		runner:    runtime,
	}
}

func TestStaticExporter(t *testing.T) {
	t.Parallel()

	t.Run("publishes passive theme pairs", func(t *testing.T) {
		t.Parallel()

		runtime := &staticExportStub{}
		exporter := newStaticExporterStub(t, runtime)
		markup := `<div data-kumbuka-plugin="com.example.diagram" data-kumbuka-module="graph"><pre><code>graph</code></pre></div>`

		output, err := exporter.materialize(context.Background(), markup, "index.md")

		require.NoError(t, err)
		assert.Equal(t, 2, runtime.calls)
		assert.NotContains(t, output, `data-kumbuka-plugin=`)
		assert.Contains(t, output, "kumbuka-static-illustration")
		assert.Contains(t, output, "text-align: left")
		assert.Contains(t, output, "/docs/assets/generated/")
		files, err := filepath.Glob(filepath.Join(exporter.outputDir, "assets", "generated", "*.svg"))
		require.NoError(t, err)
		require.Len(t, files, 2)
		for _, filename := range files {
			data, err := os.ReadFile(filename)
			require.NoError(t, err)
			assert.Equal(t, testStaticSVG, string(data))
		}
	})

	t.Run("keeps interactive blocks", func(t *testing.T) {
		t.Parallel()

		runtime := &staticExportStub{}
		exporter := newStaticExporterStub(t, runtime)
		markup := `<div data-kumbuka-plugin="com.example.diagram" data-kumbuka-module="graph"><pre><code>%% kumbuka-interactive: true</code></pre></div>`

		output, err := exporter.materialize(context.Background(), markup, "index.md")

		require.NoError(t, err)
		assert.Equal(t, markup, output)
		assert.Equal(t, 1, runtime.calls)
	})

	t.Run("ignores blocks from other plugins", func(t *testing.T) {
		t.Parallel()

		runtime := &staticExportStub{}
		exporter := newStaticExporterStub(t, runtime)
		markup := `<div data-kumbuka-plugin="com.other" data-kumbuka-module="graph"><pre><code>graph</code></pre></div>`

		output, err := exporter.materialize(context.Background(), markup, "index.md")

		require.NoError(t, err)
		assert.Equal(t, markup, output)
		assert.Zero(t, runtime.calls)
	})

	t.Run("reuses identical diagrams within a build", func(t *testing.T) {
		t.Parallel()

		runtime := &staticExportStub{}
		exporter := newStaticExporterStub(t, runtime)
		markup := `<div data-kumbuka-plugin="com.example.diagram" data-kumbuka-module="graph"><pre><code>graph</code></pre></div>`

		_, firstErr := exporter.materialize(context.Background(), markup, "first.md")
		_, secondErr := exporter.materialize(context.Background(), markup, "second.md")

		require.NoError(t, firstErr)
		require.NoError(t, secondErr)
		assert.Equal(t, 2, runtime.calls)
	})

	t.Run("rejects unsafe plugin SVG", func(t *testing.T) {
		t.Parallel()

		runtime := &staticExportStub{fail: true}
		exporter := newStaticExporterStub(t, runtime)
		markup := `<div data-kumbuka-plugin="com.example.diagram" data-kumbuka-module="graph"><pre><code>graph</code></pre></div>`

		_, err := exporter.materialize(context.Background(), markup, "index.md")

		require.ErrorContains(t, err, "index.md")
		assert.ErrorContains(t, err, "active svg element")
	})
}
