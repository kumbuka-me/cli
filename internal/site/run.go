package site

import "context"

// Result summarizes one completed static site build.
type Result struct {
	Pages     int
	OutputDir string
}

// Build builds one filesystem-backed static documentation site.
func Build(ctx context.Context, config Config) (Result, error) {
	result, err := newBuilder(staticAssets).build(ctx, config)
	if err != nil {
		return Result{}, err
	}

	return Result{Pages: result.pages, OutputDir: result.outputDir}, nil
}
