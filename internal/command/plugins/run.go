package plugins

import (
	"context"
	"fmt"
	"io"

	"github.com/kumbuka-me/cli/internal/pluginproject"
)

// RunSync downloads and validates all declared plugin packages.
func RunSync(ctx context.Context, cfg SyncConfig, stdout io.Writer) error {
	file, err := pluginproject.Load(cfg.File)
	if err != nil {
		return err
	}

	resolver, err := pluginproject.NewResolver(nil)
	if err != nil {
		return err
	}

	resolved, err := resolver.Resolve(ctx, file.Plugins)
	if err != nil {
		return err
	}
	if err := pluginproject.ValidateGraph(resolved); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Synced %d plugins from %s\n", len(resolved), cfg.File); err != nil {
		return fmt.Errorf("write sync result: %w", err)
	}
	return nil
}

// RunList prints all declared static-site plugins.
func RunList(cfg ListConfig, stdout io.Writer) error {
	file, err := pluginproject.Load(cfg.File)
	if err != nil {
		return err
	}

	for _, dependency := range file.Plugins {
		if _, err := fmt.Fprintf(
			stdout,
			"%s %s %s@%s%s\n",
			dependency.ID,
			dependency.Version,
			dependency.Repository,
			dependency.TagPrefix,
			dependency.Version,
		); err != nil {
			return fmt.Errorf("write plugin list: %w", err)
		}
	}

	return nil
}

// RunAdd validates, resolves, and adds one static-site plugin dependency.
func RunAdd(ctx context.Context, cfg AddConfig, stdout io.Writer) error {
	dependency, err := pluginproject.NormalizeDependency(pluginproject.Dependency{
		ID:         cfg.ID,
		Repository: cfg.Repository,
		TagPrefix:  cfg.TagPrefix,
		Asset:      cfg.Asset,
		Version:    cfg.Version,
	})
	if err != nil {
		return err
	}

	resolver, err := pluginproject.NewResolver(nil)
	if err != nil {
		return err
	}
	if _, err := resolver.Resolve(ctx, []pluginproject.Dependency{dependency}); err != nil {
		return err
	}
	if err := pluginproject.Add(cfg.File, dependency); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Added %s %s\n", dependency.ID, dependency.Version); err != nil {
		return fmt.Errorf("write add result: %w", err)
	}
	return nil
}

// RunRemove removes one static-site plugin dependency.
func RunRemove(cfg RemoveConfig, stdout io.Writer) error {
	if err := pluginproject.Remove(cfg.File, cfg.ID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Removed %s\n", cfg.ID); err != nil {
		return fmt.Errorf("write remove result: %w", err)
	}
	return nil
}
