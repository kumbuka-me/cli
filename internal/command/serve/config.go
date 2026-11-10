// Package serve provides the local static-site preview command.
package serve

import (
	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/cli/internal/command/build"
)

const defaultListenAddress = "127.0.0.1:8000"

// Config contains build settings and preview-server behavior.
type Config struct {
	// Build contains the effective project configuration for each preview rebuild.
	Build build.Config
	// ListenAddress specifies the local HTTP listener.
	ListenAddress string
	// NoBuild skips the initial static-site build.
	NoBuild bool
	// NoWatch disables automatic rebuilds and browser reloads.
	NoWatch bool
}

// BindFlags registers site build and preview flags and returns a reusable config resolver.
func BindFlags(flags *tinyflags.FlagSet) func() (Config, error) {
	resolveBuild := build.BindFlags(flags)
	address := flags.String("listen-address", defaultListenAddress, "Local HTTP listen address").
		NotEmpty().
		Placeholder("ADDR")
	noBuild := flags.Bool("no-build", false, "Serve existing generated files without building first")
	noWatch := flags.Bool("no-watch", false, "Disable automatic rebuilding and browser reload")

	return func() (Config, error) {
		built, err := resolveBuild()
		if err != nil {
			return Config{}, err
		}
		return Config{
			Build:         built,
			ListenAddress: *address.Value(),
			NoBuild:       *noBuild.Value(),
			NoWatch:       *noWatch.Value(),
		}, nil
	}
}
