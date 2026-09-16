// Package plugins configures and runs static-site plugin dependency commands.
package plugins

import (
	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/cli/internal/pluginproject"
)

// SyncConfig contains plugins sync settings.
type SyncConfig struct {
	File string
}

// ListConfig contains plugins list settings.
type ListConfig struct {
	File string
}

// AddConfig contains plugins add settings.
type AddConfig struct {
	File       string
	ID         string
	Repository string
	Version    string
	TagPrefix  string
	Asset      string
}

// RemoveConfig contains plugins remove settings.
type RemoveConfig struct {
	File string
	ID   string
}

// BindSyncFlags registers plugins sync flags.
func BindSyncFlags(flags *tinyflags.FlagSet) func() SyncConfig {
	file := flags.String("file", pluginproject.DefaultFile, "Plugin dependency file").
		NotEmpty().
		Placeholder("FILE")

	return func() SyncConfig {
		return SyncConfig{File: *file.Value()}
	}
}

// BindListFlags registers plugins list flags.
func BindListFlags(flags *tinyflags.FlagSet) func() ListConfig {
	file := flags.String("file", pluginproject.DefaultFile, "Plugin dependency file").
		NotEmpty().
		Placeholder("FILE")

	return func() ListConfig {
		return ListConfig{File: *file.Value()}
	}
}

// BindAddFlags registers plugins add flags.
func BindAddFlags(flags *tinyflags.FlagSet) func() AddConfig {
	file := flags.String("file", pluginproject.DefaultFile, "Plugin dependency file").
		NotEmpty().
		Placeholder("FILE")
	id := flags.String("id", "", "Plugin ID").Required().NotEmpty().Placeholder("ID")
	repository := flags.String("repository", "", "GitHub owner/repository").Required().NotEmpty().Placeholder("OWNER/REPO")
	version := flags.String("plugin-version", "", "Plugin version").Required().NotEmpty().Placeholder("VERSION")
	tagPrefix := flags.String("tag-prefix", "v", "GitHub release tag prefix").NotEmpty().Placeholder("PREFIX")
	asset := flags.String("asset", "", "Release asset base name").Placeholder("NAME")

	return func() AddConfig {
		return AddConfig{
			File:       *file.Value(),
			ID:         *id.Value(),
			Repository: *repository.Value(),
			Version:    *version.Value(),
			TagPrefix:  *tagPrefix.Value(),
			Asset:      *asset.Value(),
		}
	}
}

// BindRemoveFlags registers plugins remove flags.
func BindRemoveFlags(flags *tinyflags.FlagSet) func() RemoveConfig {
	file := flags.String("file", pluginproject.DefaultFile, "Plugin dependency file").
		NotEmpty().
		Placeholder("FILE")
	id := flags.String("id", "", "Plugin ID").Required().NotEmpty().Placeholder("ID")

	return func() RemoveConfig {
		return RemoveConfig{File: *file.Value(), ID: *id.Value()}
	}
}
