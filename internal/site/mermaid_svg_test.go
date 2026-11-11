package site

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPassiveSVG = `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg>`

func TestMaterializeMermaidSVGs(t *testing.T) {
	t.Parallel()

	t.Run("replaces passive block with content addressed light and dark images", func(t *testing.T) {
		t.Parallel()

		source := "flowchart LR\nA --> B"
		output := t.TempDir()
		createCachedMermaidPair(t, output, source)
		markup := `<div class="kumbuka-plugin-block" data-kumbuka-plugin="me.kumbuka.mermaid" data-kumbuka-module="diagrams"><pre><code class="language-mermaid">flowchart LR` + "\n" + `A --&gt; B</code></pre></div>`

		rendered, err := materializeMermaidSVGs(context.Background(), markup, "index.md", output, "/docs/")

		require.NoError(t, err)
		assert.Contains(t, rendered, `class="kumbuka-static-illustration"`)
		assert.Contains(t, rendered, `/docs/assets/diagrams/`+mermaidSVGKey(source)+`-light.svg`)
		assert.Contains(t, rendered, `/docs/assets/diagrams/`+mermaidSVGKey(source)+`-dark.svg`)
		assert.NotContains(t, rendered, `data-kumbuka-plugin`)
		assert.NotContains(t, rendered, `flowchart LR`)
	})

	t.Run("preserves interactive plugin block", func(t *testing.T) {
		t.Parallel()

		markup := `<div class="kumbuka-plugin-block" data-kumbuka-plugin="me.kumbuka.mermaid" data-kumbuka-module="diagrams"><pre><code class="language-mermaid">%% kumbuka-interactive: true` + "\n" + `flowchart LR</code></pre></div>`
		rendered, err := materializeMermaidSVGs(context.Background(), markup, "index.md", t.TempDir(), "/")

		require.NoError(t, err)
		assert.Equal(t, markup, rendered)
	})

	t.Run("retains another plugin block", func(t *testing.T) {
		t.Parallel()

		markup := `<div class="kumbuka-plugin-block" data-kumbuka-plugin="example.other" data-kumbuka-module="diagrams"><pre><code class="language-mermaid">flowchart LR</code></pre></div>`
		rendered, err := materializeMermaidSVGs(context.Background(), markup, "index.md", t.TempDir(), "/")

		require.NoError(t, err)
		assert.Equal(t, markup, rendered)
	})

	t.Run("converts multiple passive diagrams and preserves alignment", func(t *testing.T) {
		t.Parallel()

		output := t.TempDir()
		first := "%% kumbuka-align: left\nflowchart LR\nA --> B"
		second := "flowchart RL\nC --> D"
		createCachedMermaidPair(t, output, first)
		createCachedMermaidPair(t, output, second)
		markup := `<section><div data-kumbuka-plugin="me.kumbuka.mermaid" data-kumbuka-module="diagrams"><pre><code class="language-mermaid">%% kumbuka-align: left` + "\n" + `flowchart LR` + "\n" + `A --&gt; B</code></pre></div><p>Between</p><div data-kumbuka-plugin="me.kumbuka.mermaid" data-kumbuka-module="diagrams"><pre><code class="language-mermaid">flowchart RL` + "\n" + `C --&gt; D</code></pre></div></section>`

		rendered, err := materializeMermaidSVGs(context.Background(), markup, "index.md", output, "/")

		require.NoError(t, err)
		assert.Equal(t, 2, strings.Count(rendered, `class="kumbuka-static-illustration"`))
		assert.Contains(t, rendered, `text-align: left;`)
		assert.Contains(t, rendered, `<p>Between</p>`)
	})

}

func TestMermaidRenderingPolicy(t *testing.T) {
	t.Parallel()

	t.Run("recognizes interactive comment", func(t *testing.T) {
		t.Parallel()

		assert.True(t, mermaidIsInteractive("flowchart LR\n%% kumbuka-interactive: true\nA --> B"))
		assert.False(t, mermaidIsInteractive("%% kumbuka-interactive: false\nflowchart LR"))
	})

	t.Run("includes render version and settings in deterministic digest", func(t *testing.T) {
		t.Parallel()

		assert.Len(t, mermaidSVGKey("A --> B"), 64)
		assert.Equal(t, mermaidSVGKey("A --> B"), mermaidSVGKey("A --> B"))
		assert.NotEqual(t, mermaidSVGKey("A --> B"), mermaidSVGKey("B --> A"))
	})
}

func TestValidateMermaidSVG(t *testing.T) {
	t.Parallel()

	t.Run("accepts passive SVG", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, validateMermaidSVG([]byte(testPassiveSVG)))
	})

	t.Run("rejects script", func(t *testing.T) {
		t.Parallel()

		markup := `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
		require.Error(t, validateMermaidSVG([]byte(markup)))
	})

	t.Run("rejects foreign object", func(t *testing.T) {
		t.Parallel()

		markup := `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><p>unsafe</p></foreignObject></svg>`
		require.Error(t, validateMermaidSVG([]byte(markup)))
	})

	t.Run("rejects external CSS", func(t *testing.T) {
		t.Parallel()

		markup := `<svg xmlns="http://www.w3.org/2000/svg"><style>rect{fill:url(https://example.com/a)}</style></svg>`
		require.Error(t, validateMermaidSVG([]byte(markup)))
	})

	t.Run("accepts local marker reference", func(t *testing.T) {
		t.Parallel()

		markup := `<svg xmlns="http://www.w3.org/2000/svg"><path style="marker-end:url(#arrow)" d="M0 0"/></svg>`
		require.NoError(t, validateMermaidSVG([]byte(markup)))
	})
}

// createCachedMermaidPair writes valid assets so a DOM test needs no external renderer.
func createCachedMermaidPair(t *testing.T, outputDir, source string) {
	t.Helper()

	directory := filepath.Join(outputDir, "assets", "diagrams")
	require.NoError(t, os.MkdirAll(directory, 0o755))
	key := mermaidSVGKey(source)
	require.NoError(t, os.WriteFile(filepath.Join(directory, key+"-light.svg"), []byte(testPassiveSVG), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(directory, key+"-dark.svg"), []byte(testPassiveSVG), 0o644))
}
