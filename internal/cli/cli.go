// Package cli defines the standalone Kumbuka CLI command tree.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/cli/internal/command/build"
	"github.com/kumbuka-me/cli/internal/command/mirror"
	"github.com/kumbuka-me/cli/internal/command/plugins"
)

// Run parses and executes one Kumbuka CLI command.
func Run(
	ctx context.Context,
	args []string,
	version string,
	stdout, stderr io.Writer,
) error {
	root := tinyflags.NewCommand("kumbuka-cli", tinyflags.ContinueOnError).RequireCommand()
	root.Version(version)
	root.EnvPrefix("KUMBUKA_")

	buildCommand := root.Command("build", "Build a read-only static documentation site")
	resolveBuildConfig := build.BindFlags(buildCommand.FlagSet)
	buildCommand.Run(func(ctx context.Context) error {
		cfg, err := resolveBuildConfig()
		if err != nil {
			return err
		}

		return build.Run(ctx, cfg, buildCommand.OverriddenValues(), stdout)
	})

	pluginsCommand := root.Command("plugins", "Manage static-site plugin dependencies").RequireCommand()

	pluginsSyncCommand := pluginsCommand.Command("sync", "Download and validate declared plugin packages")
	resolvePluginsSyncConfig := plugins.BindSyncFlags(pluginsSyncCommand.FlagSet)
	pluginsSyncCommand.Run(func(ctx context.Context) error {
		return plugins.RunSync(ctx, resolvePluginsSyncConfig(), stdout)
	})

	pluginsListCommand := pluginsCommand.Command("list", "List declared static-site plugins")
	resolvePluginsListConfig := plugins.BindListFlags(pluginsListCommand.FlagSet)
	pluginsListCommand.Run(func(context.Context) error {
		return plugins.RunList(resolvePluginsListConfig(), stdout)
	})

	pluginsAddCommand := pluginsCommand.Command("add", "Add a static-site plugin dependency")
	resolvePluginsAddConfig := plugins.BindAddFlags(pluginsAddCommand.FlagSet)
	pluginsAddCommand.Run(func(ctx context.Context) error {
		return plugins.RunAdd(ctx, resolvePluginsAddConfig(), stdout)
	})

	pluginsRemoveCommand := pluginsCommand.Command("remove", "Remove a static-site plugin dependency")
	resolvePluginsRemoveConfig := plugins.BindRemoveFlags(pluginsRemoveCommand.FlagSet)
	pluginsRemoveCommand.Run(func(context.Context) error {
		return plugins.RunRemove(resolvePluginsRemoveConfig(), stdout)
	})

	mirrorCommand := root.Command("mirror", "Export PostgreSQL content as a Git-friendly Markdown mirror")
	resolveMirrorConfig := mirror.BindFlags(mirrorCommand.FlagSet)
	mirrorCommand.Run(func(ctx context.Context) error {
		return mirror.Run(ctx, resolveMirrorConfig(), stdout)
	})

	runner, err := root.ParseRunner(args)
	if err != nil {
		switch {
		case tinyflags.IsHelpRequested(err), tinyflags.IsVersionRequested(err):
			_, _ = fmt.Fprint(stdout, err.Error())
			return nil
		case tinyflags.IsCommandRequired(err):
			help, _ := tinyflags.HelpText(err)
			_, _ = fmt.Fprint(stderr, help)
			return nil
		default:
			_, _ = fmt.Fprintln(stderr, err)
			return err
		}
	}

	if err := runner.Run(ctx); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return err
	}

	return nil
}
