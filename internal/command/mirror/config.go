// Package mirror configures and runs the database mirror command.
package mirror

import (
	appkitconfig "github.com/containeroo/appkit/config"
	"github.com/containeroo/tinyflags"

	appkitlogging "github.com/containeroo/appkit/logging"
)

const defaultOutputDir = "kumbuka-mirror"

// Config contains database mirror command settings.
type Config struct {
	// DatabaseURL identifies the PostgreSQL source database.
	DatabaseURL string
	// OutputDir receives the generated mirror snapshot.
	OutputDir string
	// LogFormat selects text or JSON command logging.
	LogFormat appkitlogging.LogFormat
}

// BindFlags registers mirror flags and returns the parsed configuration.
func BindFlags(flags *tinyflags.FlagSet) func() Config {
	flags.EnvPrefix("KUMBUKA_")

	cfg := Config{
		OutputDir: defaultOutputDir,
		LogFormat: appkitlogging.LogFormatText,
	}

	flags.StringVar(&cfg.DatabaseURL, "database-url", "", "PostgreSQL connection URL").
		Required().
		Placeholder("URL").
		OverriddenValueMaskFn(tinyflags.MaskPostgresURL).
		Value()
	flags.StringVar(&cfg.OutputDir, "output", defaultOutputDir, "Directory that receives the Git-friendly mirror").
		Placeholder("DIR").
		Value()
	finishLogFormat := appkitconfig.RegisterLogFormat(flags, &cfg.LogFormat)

	return func() Config {
		finishLogFormat()
		return cfg
	}
}
