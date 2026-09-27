// Package mirror configures and runs the database mirror command.
package mirror

import (
	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/kumbuka/pkg/logging"
)

const defaultOutputDir = "kumbuka-mirror"

// Config contains database mirror command settings.
type Config struct {
	// DatabaseURL identifies the PostgreSQL source database.
	DatabaseURL string
	// OutputDir receives the generated mirror snapshot.
	OutputDir string
	// LogFormat selects text or JSON command logging.
	LogFormat logging.LogFormat
}

// BindFlags registers mirror flags and returns the parsed configuration.
func BindFlags(flags *tinyflags.FlagSet) func() Config {
	flags.EnvPrefix("KUMBUKA_")

	cfg := Config{
		OutputDir: defaultOutputDir,
		LogFormat: logging.LogFormatText,
	}

	flags.StringVar(&cfg.DatabaseURL, "database-url", "", "PostgreSQL connection URL").
		Required().
		Placeholder("URL").
		OverriddenValueMaskFn(tinyflags.MaskPostgresURL).
		Value()
	flags.StringVar(&cfg.OutputDir, "output", defaultOutputDir, "Directory that receives the Git-friendly mirror").
		Placeholder("DIR").
		Value()
	logFormat := tinyflags.Enum(
		flags,
		"log-format",
		logging.LogFormatJSON,
		"Log output format",
		logging.LogFormatText,
		logging.LogFormatJSON,
	).
		Short("l").
		Placeholder("FORMAT").
		Value()

	return func() Config {
		cfg.LogFormat = *logFormat
		return cfg
	}
}
