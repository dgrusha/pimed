package spa

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":               {Data: []byte("<html>app</html>")},
		"favicon.svg":              {Data: []byte("<svg/>")},
		"_app/immutable/entry.js":  {Data: []byte("console.log(1)")},
		"_app/immutable/style.css": {Data: []byte("body{}")},
	}
	h := Handler(fsys)

	tests := []struct {
		name, method, path string
		wantStatus         int
		wantBody           string
		wantType           string
		wantCache          string
	}{
		{"root", "GET", "/", 200, "app", "text/html; charset=utf-8", "no-cache"},
		{"index explicit", "GET", "/index.html", 200, "app", "text/html; charset=utf-8", "no-cache"},
		{"fallback", "GET", "/files/some/deep/route", 200, "app", "text/html; charset=utf-8", "no-cache"},
		{"traversal cleaned", "GET", "/../../etc/passwd", 200, "app", "text/html; charset=utf-8", "no-cache"},
		{"asset", "GET", "/favicon.svg", 200, "<svg/>", "image/svg+xml", "no-cache"},
		{"immutable js", "GET", "/_app/immutable/entry.js", 200, "console", "text/javascript; charset=utf-8", "public, max-age=31536000, immutable"},
		{"immutable css", "GET", "/_app/immutable/style.css", 200, "body", "text/css; charset=utf-8", "public, max-age=31536000, immutable"},
		{"missing app asset", "GET", "/_app/immutable/gone.js", 404, "", "", ""},
		{"directory falls back", "GET", "/_app", 200, "app", "text/html; charset=utf-8", "no-cache"},
		{"post rejected", "POST", "/", 405, "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q", rec.Body.String())
			}
			if tt.wantType != "" && rec.Header().Get("Content-Type") != tt.wantType {
				t.Fatalf("Content-Type = %q", rec.Header().Get("Content-Type"))
			}
			if tt.wantCache != "" && rec.Header().Get("Cache-Control") != tt.wantCache {
				t.Fatalf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestHandlerNotBuilt(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(fstest.MapFS{}).ServeHTTP(rec, httptest.NewRequest("GET", "/anything", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "has not been built") {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestContentSecurityPolicy(t *testing.T) {
	// The inline body uses CRLF; browsers hash it with LF line endings.
	index := "<html><head><script type=\"module\" src=\"/_app/x.js\"></script></head>" +
		"<body><SCRIPT>alert(1)\r\n</SCRIPT></body></html>"
	h := Handler(fstest.MapFS{"index.html": {Data: []byte(index)}})
	for _, p := range []string{"/", "/missing/route", "/_app/immutable/gone.js"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		csp := rec.Header().Get("Content-Security-Policy")
		want := "script-src 'self' 'sha256-MaeD7tQk/YNyd6Pm9J12ROmv0Z93QwNN4VH7v3gI+RI='; "
		if !strings.Contains(csp, want) || !strings.Contains(csp, "frame-ancestors 'none'") ||
			!strings.HasPrefix(csp, "default-src 'self'; ") {
			t.Fatalf("%s: CSP = %q", p, csp)
		}
	}

	// Without a build there is nothing inline to allow.
	csp := ContentSecurityPolicy(fstest.MapFS{})
	if !strings.Contains(csp, "script-src 'self'; ") {
		t.Fatalf("CSP = %q", csp)
	}
}
