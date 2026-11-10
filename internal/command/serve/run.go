package serve

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kumbuka-me/cli/internal/preview"
	"github.com/kumbuka-me/cli/internal/site"
)

// Run builds and serves a local static-site preview until ctx is canceled.
func Run(ctx context.Context, resolve func() (Config, error), stdout, stderr io.Writer) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	config, err := resolve()
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen for preview: %w", err)
	}
	defer listener.Close() // nolint:errcheck

	basePath, err := site.BasePath(config.Build.Site.SiteURL)
	if err != nil {
		return err
	}
	previewURL := "http://" + publicAddress(listener.Addr()) + basePath
	if !config.NoBuild {
		if err := buildPreview(ctx, config, previewURL, stdout); err != nil {
			return err
		}
	} else if err := checkOutput(config.Build.Site.OutputDir); err != nil {
		return err
	}

	var hub *preview.Hub
	if !config.NoWatch {
		hub = &preview.Hub{}
	}
	var servingDir atomic.Value
	servingDir.Store(config.Build.Site.OutputDir)
	server := &http.Server{
		Handler:           preview.Handler(func() string { return servingDir.Load().(string) }, basePath, hub),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()

	if _, err := fmt.Fprintf(stdout, "Serving: %s\n", previewURL); err != nil {
		if hub != nil {
			hub.Close()
		}
		_ = server.Shutdown(context.Background())
		<-serverDone // nolint:errcheck
		return err
	}
	if hub != nil {
		if _, err := fmt.Fprintln(stdout, "Watching for changes..."); err != nil {
			hub.Close()
			_ = server.Shutdown(context.Background())
			<-serverDone // nolint:errcheck
			return err
		}
	}

	if _, err := fmt.Fprintln(stdout, "Press Ctrl+C to stop."); err != nil {
		if hub != nil {
			hub.Close()
		}
		_ = server.Shutdown(context.Background())
		<-serverDone // nolint:errcheck
		return err
	}

	watchDone := make(chan struct{})
	if hub != nil {
		go func() {
			defer close(watchDone)
			current := config
			preview.Watch(ctx, func() []string {
				return preview.WatchPaths(current.Build.Site, current.Build.ConfigPath)
			}, func() {
				updated, err := resolve()
				if err != nil {
					_, _ = fmt.Fprintf(stderr, "Preview configuration error: %v\n", err)
					return
				}
				current = updated
				if err := buildPreview(ctx, updated, previewURL, stdout); err != nil {
					_, _ = fmt.Fprintf(stderr, "Preview rebuild failed: %v\n", err)
					return
				}
				servingDir.Store(updated.Build.Site.OutputDir)
				hub.Notify()
			}, func(err error) {
				_, _ = fmt.Fprintf(stderr, "Preview watcher error: %v\n", err)
			})
		}()
	} else {
		close(watchDone)
	}

	select {
	case <-ctx.Done():
	case err := <-serverDone:
		stop()
		<-watchDone
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve preview: %w", err)
		}
		return nil
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if hub != nil {
		hub.Close()
	}
	err = server.Shutdown(shutdown)
	<-serverDone // nolint:errcheck
	<-watchDone
	return err
}

// buildPreview generates the site's assets with a local origin and the configured public path.
func buildPreview(ctx context.Context, config Config, previewURL string, stdout io.Writer) error {
	buildConfig := config.Build.Site
	buildConfig.SiteURL = previewURL
	result, err := site.Build(ctx, buildConfig)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "Built %d pages into %s\n", result.Pages, result.OutputDir)
	return err
}

// checkOutput rejects missing or incomplete generated sites before the preview starts.
func checkOutput(directory string) error {
	info, err := os.Stat(filepath.Join(directory, "index.html"))
	if err != nil {
		return fmt.Errorf("preview output not built (run kumbuka-cli build): %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("preview output is missing a regular index.html")
	}
	return nil
}

// publicAddress converts a listener address into a browser-reachable local address.
func publicAddress(address net.Addr) string {
	host, port, err := net.SplitHostPort(address.String())
	if err != nil {
		return address.String()
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if host == "" {
		host = "127.0.0.1"
	} else if ip != nil && ip.IsUnspecified() {
		if ip.To4() == nil {
			host = "::1"
		} else {
			host = "127.0.0.1"
		}
	}
	return net.JoinHostPort(host, port)
}
