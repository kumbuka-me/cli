package site

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/kumbuka-me/sdk/staticexport"
)

//go:embed static_export_runner.mjs
var staticExportRunnerSource []byte

// staticExportRuntime renders one browser plugin block in an isolated local build environment.
type staticExportRuntime interface {
	Render(context.Context, string, string, string, string) (*staticexport.Result, error)
	Close() error
}

// browserExportRunner keeps one headless browser process alive across a static-site build.
type browserExportRunner struct {
	// process owns the Node-based Playwright browser worker.
	process *exec.Cmd
	// input sends one JSON request per rendered block and color scheme.
	input io.WriteCloser
	// output decodes bounded artifact responses returned by the worker.
	output *json.Decoder
	// stderr preserves diagnostics if the worker exits unexpectedly.
	stderr bytes.Buffer
	// scriptPath is the temporary, embedded runner script.
	scriptPath string
}

// staticExportRequest identifies one exact package asset and the plugin block to render.
type staticExportRequest struct {
	// AssetDirectory contains only the copied package's browser assets.
	AssetDirectory string `json:"asset_directory"`
	// JavaScript names the browser-module script in that asset directory.
	JavaScript string `json:"javascript"`
	// Markup is the sanitized host-rendered plugin wrapper.
	Markup string `json:"markup"`
	// Theme selects the build output variant.
	Theme string `json:"theme"`
}

// staticExportResponse carries a generated artifact or an explicit browser error.
type staticExportResponse struct {
	// Result is nil when the plugin declines to export this block.
	Result *staticexport.Result `json:"result"`
	// Error describes a failed export without terminating the worker.
	Error string `json:"error,omitempty"`
}

// Render invokes an opted-in browser module with no external network access.
func (r *browserExportRunner) Render(ctx context.Context, assets, script, markup, theme string) (*staticexport.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.process == nil {
		if err := r.start(ctx); err != nil {
			return nil, err
		}
	}

	request := staticExportRequest{
		AssetDirectory: assets,
		JavaScript:     script,
		Markup:         markup,
		Theme:          theme,
	}
	if err := json.NewEncoder(r.input).Encode(request); err != nil {
		return nil, fmt.Errorf("send static export request: %w (%s)", err, r.stderr.String())
	}

	var response staticExportResponse
	if err := r.output.Decode(&response); err != nil {
		return nil, fmt.Errorf("read browser static export: %w (%s)", err, r.stderr.String())
	}
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	return response.Result, nil
}

// start materializes the bundled browser worker and launches Node when the first export is needed.
func (r *browserExportRunner) start(ctx context.Context) error {
	script, err := os.CreateTemp("", "kumbuka-static-export-*.mjs")
	if err != nil {
		return fmt.Errorf("create browser export runner: %w", err)
	}
	r.scriptPath = script.Name()
	if _, err := script.Write(staticExportRunnerSource); err != nil {
		_ = script.Close()
		_ = os.Remove(r.scriptPath)
		return err
	}
	if err := script.Close(); err != nil {
		_ = os.Remove(r.scriptPath)
		return err
	}

	command := exec.CommandContext(ctx, "node", r.scriptPath)
	command.Stderr = &r.stderr
	input, err := command.StdinPipe()
	if err != nil {
		_ = os.Remove(r.scriptPath)
		return fmt.Errorf("open browser export input: %w", err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		_ = input.Close()
		_ = os.Remove(r.scriptPath)
		return fmt.Errorf("open browser export output: %w", err)
	}
	if err := command.Start(); err != nil {
		_ = input.Close()
		_ = os.Remove(r.scriptPath)
		return fmt.Errorf("start browser export (requires Node.js and the Playwright package): %w", err)
	}
	r.process = command
	r.input = input
	r.output = json.NewDecoder(output)
	return nil
}

// Close flushes and stops the shared worker, cleaning up its temporary script.
func (r *browserExportRunner) Close() error {
	if r.process == nil {
		return nil
	}
	inputErr := r.input.Close()
	waitErr := r.process.Wait()
	removeErr := os.Remove(r.scriptPath)
	if waitErr != nil {
		return fmt.Errorf("static export browser failed: %w: %s", waitErr, strings.TrimSpace(r.stderr.String()))
	}
	return errors.Join(inputErr, removeErr)
}
