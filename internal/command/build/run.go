package build

import (
	"context"
	"fmt"
	"io"

	"github.com/kumbuka-me/cli/internal/site"
	"github.com/kumbuka-me/kumbuka/pkg/logging"
)

// Run builds one static documentation site.
func Run(ctx context.Context, cfg Config, overrides map[string]any, stdout io.Writer) error {
	logger := logging.Setup(cfg.LogFormat, false, stdout)
	setupLogger := logger.With("component", "setup")

	if len(overrides) > 0 {
		setupLogger.Info(
			"CLI Overrides",
			"event", "cli_overrides",
			"overrides", overrides,
		)
	}

	result, err := site.Build(ctx, cfg.Site)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Built %d pages into %s\n", result.Pages, result.OutputDir); err != nil {
		return fmt.Errorf("write build result: %w", err)
	}
	return nil
}
