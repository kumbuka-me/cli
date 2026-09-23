package site

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareSearchEntries(t *testing.T) {
	t.Parallel()

	t.Run("orders titles case insensitively", func(t *testing.T) {
		t.Parallel()

		left := searchEntry{Title: "Alpha", URL: "/alpha/"}
		right := searchEntry{Title: "beta", URL: "/beta/"}

		assert.Negative(t, compareSearchEntries(left, right))
	})

	t.Run("breaks case-insensitive title ties by original title", func(t *testing.T) {
		t.Parallel()

		left := searchEntry{Title: "API", URL: "/z/"}
		right := searchEntry{Title: "api", URL: "/a/"}

		assert.Negative(t, compareSearchEntries(left, right))
	})

	t.Run("breaks exact title ties by URL", func(t *testing.T) {
		t.Parallel()

		left := searchEntry{Title: "Guide", URL: "/a/"}
		right := searchEntry{Title: "Guide", URL: "/b/"}

		assert.Negative(t, compareSearchEntries(left, right))
	})
}
