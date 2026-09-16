package site

import (
	"testing"

	md "github.com/kumbuka-me/kumbuka/pkg/markdown"
)

// testMarkdownRenderer returns a renderer without starting a plugin runtime.
func testMarkdownRenderer(t testing.TB) *md.Renderer {
	t.Helper()
	return md.NewWithRegistry(nil)
}
