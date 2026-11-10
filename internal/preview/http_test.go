package preview

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandlerInjectsReloadOnlyIntoStaticSitePages(t *testing.T) {
	t.Parallel()

	output := t.TempDir()
	if err := os.WriteFile(filepath.Join(output, "index.html"), []byte(`<html><body class="static-site">Home</body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(output, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "plugins", "frame.html"), []byte(`<html><body>Frame</body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(Handler(func() string { return output }, "/docs/", &Hub{}))
	defer server.Close()

	for _, item := range []struct {
		path       string
		scriptWant bool
	}{
		{path: "/docs/", scriptWant: true},
		{path: "/docs/plugins/frame.html", scriptWant: false},
	} {
		response, err := server.Client().Get(server.URL + item.path)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", item.path, response.StatusCode)
		}
		if strings.Contains(string(data), `/docs/__kumbuka/reload.js`) != item.scriptWant {
			t.Fatalf("%s: unexpected preview injection: %s", item.path, data)
		}
	}
}

func TestHandlerRedirectsToSitePrefix(t *testing.T) {
	t.Parallel()

	handler := Handler(func() string { return t.TempDir() }, "/docs/", nil)
	request := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != "/docs/" {
		t.Fatalf("status %d, location %s", response.Code, response.Header().Get("Location"))
	}
}

func TestEventsNotifyConnectedBrowser(t *testing.T) {
	t.Parallel()

	hub := &Hub{}
	server := httptest.NewServer(Handler(func() string { return t.TempDir() }, "/", hub))
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/__kumbuka/reload")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected content type: %s", response.Header.Get("Content-Type"))
	}

	lines := make(chan string, 4)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	hub.Notify()
	deadline := time.After(time.Second)
	for {
		select {
		case line := <-lines:
			if line == "event: reload" {
				return
			}
		case <-deadline:
			t.Fatal("no reload event received")
		}
	}
}
