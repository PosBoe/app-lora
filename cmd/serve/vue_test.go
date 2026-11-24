package serve

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestVueHandler(t *testing.T) {
	distFS := fstest.MapFS{
		"index.html":      {Data: []byte("<html>index</html>")},
		"assets/app-1.js": {Data: []byte("console.log(1)")},
	}
	h := newVueHandler(distFS)

	tests := []struct {
		name      string
		path      string
		wantCode  int
		wantBody  string
		wantCache string
	}{
		{name: "root", path: "/", wantCode: http.StatusOK, wantBody: "<html>index</html>"},
		{name: "asset", path: "/assets/app-1.js", wantCode: http.StatusOK, wantBody: "console.log(1)", wantCache: "public, max-age=31536000, immutable"},
		{name: "spa fallback", path: "/projects/42", wantCode: http.StatusOK, wantBody: "<html>index</html>"},
		{name: "unknown api", path: "/api/v1/nope", wantCode: http.StatusNotFound},
		{name: "well-known", path: "/.well-known/security.txt", wantCode: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, rec.Body.String())
			}
			assert.Equal(t, tt.wantCache, rec.Header().Get("Cache-Control"))
		})
	}
}
