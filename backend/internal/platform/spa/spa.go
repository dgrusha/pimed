// Package spa serves the embedded single-page application.
package spa

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
)

const (
	indexFile     = "index.html"
	immutablePath = "_app/immutable/"
	appPath       = "_app/"
)

const notBuiltPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>PiMed</title></head>
<body><h1>PiMed</h1><p>The frontend has not been built. Build it into
<code>backend/web/dist/</code> and rebuild the server. The API is available under <code>/api</code>.</p></body></html>
`

// Handler serves files from fsys. Unknown paths fall back to index.html
// (client-side routing), except missing files under _app/, which 404. If
// index.html itself is missing, a placeholder page is served. Every response
// carries the Content-Security-Policy from ContentSecurityPolicy(fsys).
func Handler(fsys fs.FS) http.Handler {
	csp := ContentSecurityPolicy(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", csp)
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = indexFile
		}
		if serveFile(w, r, fsys, name) {
			return
		}
		if strings.HasPrefix(name, appPath) {
			http.NotFound(w, r)
			return
		}
		if serveFile(w, r, fsys, indexFile) {
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		io.WriteString(w, notBuiltPage)
	})
}

// serveFile serves name if it is a regular file in fsys and reports whether
// it did.
func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return false
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		return false
	}
	h := w.Header()
	if ct := contentType(name); ct != "" {
		h.Set("Content-Type", ct)
	}
	if strings.HasPrefix(name, immutablePath) {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "no-cache")
	}
	http.ServeContent(w, r, name, st.ModTime(), rs)
	return true
}

// contentType pins types for common web assets so results do not depend on
// the host's MIME database (e.g. the Windows registry mapping .js oddly).
func contentType(name string) string {
	switch ext := strings.ToLower(path.Ext(name)); ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json"
	case ".webmanifest":
		return "application/manifest+json"
	case ".svg":
		return "image/svg+xml"
	case ".wasm":
		return "application/wasm"
	case ".woff2":
		return "font/woff2"
	default:
		return mime.TypeByExtension(ext)
	}
}

// cspTemplate is the SPA policy; %s receives the inline-script hashes. Keep
// it in sync with kit.csp in frontend/svelte.config.js (which additionally
// emits it as a <meta> tag; frame-ancestors only works as a header).
const cspTemplate = "default-src 'self'; script-src 'self'%s; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; media-src 'self' blob:; connect-src 'self'; font-src 'self'; " +
	"object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// inlineScript matches <script> elements; group 1 is the attributes, group 2
// the body.
var inlineScript = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script\s*>`)

var srcAttr = regexp.MustCompile(`(?i)(^|\s)src\s*=`)

// ContentSecurityPolicy returns the CSP for SPA responses. SvelteKit's
// index.html boots the app from an inline <script>; its SHA-256 is computed
// here from the embedded file at startup, so the header always matches the
// build without copying hashes around. The result is the same hash SvelteKit
// puts into its <meta> CSP.
func ContentSecurityPolicy(fsys fs.FS) string {
	var hashes strings.Builder
	if b, err := fs.ReadFile(fsys, indexFile); err == nil {
		for _, h := range InlineScriptHashes(string(b)) {
			hashes.WriteString(" '" + h + "'")
		}
	}
	return strings.Replace(cspTemplate, "%s", hashes.String(), 1)
}

// InlineScriptHashes returns CSP source expressions ("sha256-...") for every
// inline <script> in doc. Line endings are normalized the way the HTML parser
// does before browsers hash script text.
func InlineScriptHashes(doc string) []string {
	var out []string
	for _, m := range inlineScript.FindAllStringSubmatch(doc, -1) {
		if srcAttr.MatchString(m[1]) {
			continue
		}
		body := strings.ReplaceAll(m[2], "\r\n", "\n")
		body = strings.ReplaceAll(body, "\r", "\n")
		sum := sha256.Sum256([]byte(body))
		out = append(out, "sha256-"+base64.StdEncoding.EncodeToString(sum[:]))
	}
	return out
}
