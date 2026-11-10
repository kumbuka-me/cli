package serve

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kumbuka-me/cli/internal/command/build"
	"github.com/kumbuka-me/cli/internal/site"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunServesExistingFilesWithoutBuild(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "site")
	require.NoError(t, os.Mkdir(output, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(output, "index.html"), []byte("<h1>Preview</h1>"), 0o644))
	config := site.DefaultConfig()
	config.OutputDir = output
	config.SiteURL = "https://example.test/docs/"
	options := Config{Build: build.Config{Site: config}, ListenAddress: "127.0.0.1:0", NoBuild: true, NoWatch: true}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var outputText lockedBuffer
	finished := make(chan error, 1)
	go func() { finished <- Run(ctx, func() (Config, error) { return options, nil }, &outputText, &outputText) }()

	var address string
	deadline := time.After(5 * time.Second)
	for address == "" {
		select {
		case err := <-finished:
			t.Fatalf("server stopped: %v", err)
		case <-deadline:
			t.Fatalf("server not ready: %s", outputText.String())
		case <-time.After(10 * time.Millisecond):
			if _, urlLine, ok := strings.Cut(outputText.String(), "Serving: "); ok {
				address, _, _ = strings.Cut(urlLine, "\n")
				address = strings.TrimSpace(address)
			}
		}
	}

	response, err := http.Get(address)
	require.NoError(t, err)
	defer response.Body.Close() // nolint:errcheck
	assert.Equal(t, http.StatusOK, response.StatusCode)
	cancel()
	require.NoError(t, <-finished)
}

func TestRunRejectsUnbuiltOutput(t *testing.T) {
	config := site.DefaultConfig()
	config.OutputDir = filepath.Join(t.TempDir(), "missing")
	options := Config{Build: build.Config{Site: config}, ListenAddress: "127.0.0.1:0", NoBuild: true, NoWatch: true}
	err := Run(context.Background(), func() (Config, error) { return options, nil }, &bytes.Buffer{}, &bytes.Buffer{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "run kumbuka-cli build")
}

func TestPublicAddress(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ input, expected string }{
		{":8000", "127.0.0.1:8000"},
		{"0.0.0.0:8080", "127.0.0.1:8080"},
		{"127.0.0.1:8000", "127.0.0.1:8000"},
	} {
		t.Run(fmt.Sprint(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, publicAddress(fakeAddr(tt.input)))
		})
	}
}

// lockedBuffer makes server output safe to inspect from another test goroutine.
type lockedBuffer struct {
	// mu serializes access from the test and server goroutines.
	mu sync.Mutex
	// buffer accumulates status output for assertions.
	buffer bytes.Buffer
}

// Write appends output while holding the test buffer lock.
func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

// String returns the current output while holding the test buffer lock.
func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// fakeAddr implements net.Addr for local address normalization tests.
type fakeAddr string

func (fakeAddr) Network() string  { return "tcp" }
func (a fakeAddr) String() string { return string(a) }
