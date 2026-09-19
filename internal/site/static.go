package site

import (
	"embed"
	"io/fs"
)

//go:embed static
var embeddedStatic embed.FS

var staticAssets = mustSubStatic(embeddedStatic, "static")

// mustSubStatic returns the embedded static subtree or panics for a broken build-time embed.
func mustSubStatic(root fs.FS, directory string) fs.FS {
	assets, err := fs.Sub(root, directory)
	if err != nil {
		panic(err)
	}
	return assets
}
