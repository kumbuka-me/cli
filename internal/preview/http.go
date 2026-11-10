// Package preview serves generated static sites locally and notifies browsers about rebuilds.
package preview

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"
)

const reloadScript = `(function () {
  const script = document.currentScript;
  if (!script || !window.EventSource) return;
  const events = new EventSource(new URL("reload", script.src));
  events.addEventListener("reload", () => window.location.reload());
  window.addEventListener("pagehide", () => events.close(), {once: true});
})();
`

// Hub broadcasts a successful rebuild to connected preview browsers.
type Hub struct {
	// mu protects clients and the closed state.
	mu sync.Mutex
	// clients contains all active browser event streams.
	clients map[chan struct{}]struct{}
	// closed prevents subscribing after shutdown.
	closed bool
}

// Notify asks every connected preview browser to reload.
func (h *Hub) Notify() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client <- struct{}{}:
		default:
		}
	}
}

// Close disconnects every browser and prevents new event subscriptions.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for client := range h.clients {
		close(client)
		delete(h.clients, client)
	}
}

// subscribe registers a browser and returns a cleanup function.
func (h *Hub) subscribe() (<-chan struct{}, func()) {
	client := make(chan struct{}, 1)
	h.mu.Lock()
	if h.closed {
		close(client)
		h.mu.Unlock()
		return client, func() {}
	}
	if h.clients == nil {
		h.clients = make(map[chan struct{}]struct{})
	}
	h.clients[client] = struct{}{}
	h.mu.Unlock()
	return client, func() {
		h.mu.Lock()
		delete(h.clients, client)
		h.mu.Unlock()
	}
}

// Handler serves generated files beneath basePath and optionally injects the reload client.
func Handler(outputDir func() string, basePath string, hub *Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if basePath != "/" && (r.URL.Path == "/" || r.URL.Path == strings.TrimSuffix(basePath, "/")) {
			http.Redirect(w, r, basePath, http.StatusMovedPermanently)
			return
		}
		if !strings.HasPrefix(r.URL.Path, basePath) {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, basePath)
		if hub != nil {
			switch path {
			case "__kumbuka/reload.js":
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
				_, _ = fmt.Fprint(w, reloadScript)
				return
			case "__kumbuka/reload":
				serveEvents(w, r, hub)
				return
			}
		}

		files := http.FileServer(http.Dir(outputDir()))
		request := r.Clone(r.Context())
		request.URL.Path = "/" + path
		request.URL.RawPath = ""
		if hub == nil || r.Method != http.MethodGet || (!strings.HasSuffix(path, "/") && !strings.HasSuffix(path, ".html") && path != "") {
			files.ServeHTTP(w, request)
			return
		}

		// Inject only into preview responses; the generated files remain deployment-ready.
		recorded := &bufferedResponse{header: make(http.Header)}
		files.ServeHTTP(recorded, request)
		body := recorded.body.Bytes()
		if recorded.status == http.StatusOK && bytes.Contains(body, []byte(`<body class="static-site"`)) {
			script := []byte(`<script src="` + html.EscapeString(basePath+"__kumbuka/reload.js") + `"></script>`)
			body = bytes.Replace(body, []byte("</body>"), append(script, []byte("</body>")...), 1)
		}
		for key, values := range recorded.header {
			if strings.EqualFold(key, "Content-Length") {
				continue
			}
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(recorded.status)
		_, _ = w.Write(body)
	})
}

// bufferedResponse captures an HTML response before local-only script injection.
type bufferedResponse struct {
	// header contains the upstream file server's response headers.
	header http.Header
	// body contains the unmodified generated HTML before injection.
	body bytes.Buffer
	// status is the recorded HTTP response code.
	status int
}

// Header contains headers accumulated by the file server.
func (w *bufferedResponse) Header() http.Header { return w.header }

// WriteHeader captures the first response status.
func (w *bufferedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

// Write collects the response body for subsequent injection.
func (w *bufferedResponse) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(data)
}

// serveEvents streams rebuild notifications over the browser's EventSource connection.
func serveEvents(w http.ResponseWriter, r *http.Request, hub *Hub) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	client, unsubscribe := hub.subscribe()
	defer unsubscribe()
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case _, open := <-client:
			if !open {
				return
			}
			_, _ = fmt.Fprint(w, "event: reload\ndata: updated\n\n")
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
		}
		flusher.Flush()
	}
}
