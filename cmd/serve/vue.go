package serve

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// vueHandler serves the built SPA from vue/dist. A path the dist does not
// hold gets index.html, so the client router handles it.
type vueHandler struct {
	distFS     fs.FS
	fileServer http.Handler
}

func newVueHandler(distFS fs.FS) *vueHandler {
	return &vueHandler{
		distFS:     distFS,
		fileServer: http.FileServerFS(distFS),
	}
}

func (h *vueHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	urlPath := strings.TrimPrefix(req.URL.Path, "/")

	if urlPath != "" && h.pathExists(urlPath) {
		switch path.Ext(urlPath) {
		case "", ".html":
			// index pages are not cached
		default:
			// Vite puts a content hash in every asset name.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		h.fileServer.ServeHTTP(w, req)
		return
	}

	if isNotFoundPath(urlPath) {
		http.NotFound(w, req)
		return
	}

	// Copy the request: the handler must not change the caller's URL.
	req2 := req.Clone(req.Context())
	req2.URL.Path = "/"
	h.fileServer.ServeHTTP(w, req2)
}

// isNotFoundPath reports whether an unknown path gets a 404 instead of the
// SPA. An API path is one. So is a well-known URI (RFC 8615). urlPath has no
// leading slash.
func isNotFoundPath(urlPath string) bool {
	return strings.HasPrefix(urlPath, ".well-known/") ||
		strings.HasPrefix(urlPath, "api/")
}

func (h *vueHandler) pathExists(name string) bool {
	f, err := h.distFS.Open(name)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
