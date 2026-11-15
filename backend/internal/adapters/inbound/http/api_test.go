package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	httpapi "pimed/internal/adapters/inbound/http"
	"pimed/internal/adapters/outbound/argon2id"
	"pimed/internal/adapters/outbound/sqlite"
	"pimed/internal/adapters/outbound/token"
	"pimed/internal/app"
)

// client is a tiny test client that tracks the session cookie like a browser.
type client struct {
	t       *testing.T
	h       http.Handler
	session string
}

type response struct {
	status int
	header http.Header
	body   map[string]any
	raw    string
}

func (c *client) do(method, path, body string, hdr map[string]string) response {
	c.t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.RemoteAddr = "192.0.2.1:1234"
	if method != http.MethodGet && hdr == nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	if c.session != "" {
		req.AddCookie(&http.Cookie{Name: httpapi.SessionCookie, Value: c.session})
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	res := response{status: rec.Code, header: rec.Header(), raw: rec.Body.String()}
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(rec.Body.Bytes(), &res.body); err != nil {
			c.t.Fatalf("%s %s: bad JSON %q: %v", method, path, res.raw, err)
		}
	}
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == httpapi.SessionCookie {
			if ck.MaxAge < 0 || ck.Value == "" {
				c.session = ""
			} else {
				c.session = ck.Value
			}
		}
	}
	return res
}

func (c *client) expect(method, path, body string, want int) response {
	c.t.Helper()
	res := c.do(method, path, body, nil)
	if res.status != want {
		c.t.Fatalf("%s %s: status %d, want %d (body %s)", method, path, res.status, want, res.raw)
	}
	return res
}

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	return newHandlerWith(t, httpapi.Options{CookieSecure: true})
}

// newHandlerWith builds a handler backed by a fresh database; opts.SPA
// defaults to a stub answering "spa".
func newHandlerWith(t *testing.T, opts httpapi.Options) http.Handler {
	t.Helper()
	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc, err := app.New(app.Deps{
		Users:      sqlite.NewUserRepository(db),
		Sessions:   sqlite.NewSessionRepository(db),
		Hasher:     argon2id.New(argon2id.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32}),
		Tokens:     token.Generator{},
		SessionTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.SPA == nil {
		opts.SPA = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "spa") })
	}
	return httpapi.NewHandler(svc, opts)
}

func userField(t *testing.T, res response, field string) any {
	t.Helper()
	u, ok := res.body["user"].(map[string]any)
	if !ok {
		t.Fatalf("no user in %s", res.raw)
	}
	return u[field]
}

func TestFullFlow(t *testing.T) {
	h := newHandler(t)
	admin := &client{t: t, h: h}

	// Health and setup status.
	if res := admin.expect("GET", "/api/health", "", 200); res.body["status"] != "ok" {
		t.Fatalf("health body %s", res.raw)
	}
	if res := admin.expect("GET", "/api/setup/status", "", 200); res.body["needsSetup"] != true {
		t.Fatalf("status body %s", res.raw)
	}

	// Setup validation, then success.
	admin.expect("POST", "/api/setup", `{"username":"x","password":"password1"}`, 400)
	admin.expect("POST", "/api/setup", `{"username":"Admin","password":"password1","extra":1}`, 400)
	res := admin.expect("POST", "/api/setup", `{"username":"Admin","password":"password1"}`, 201)
	if userField(t, res, "username") != "admin" || userField(t, res, "role") != "admin" || userField(t, res, "id") != 1.0 {
		t.Fatalf("setup body %s", res.raw)
	}
	if _, err := time.Parse(time.RFC3339, userField(t, res, "createdAt").(string)); err != nil {
		t.Fatalf("createdAt: %v", err)
	}
	if _, ok := res.body["user"].(map[string]any)["passwordHash"]; ok {
		t.Fatal("password hash leaked")
	}
	setCookie := res.header.Get("Set-Cookie")
	for _, attr := range []string{"pimed_session=", "Path=/", "HttpOnly", "Secure", "SameSite=Strict"} {
		if !strings.Contains(setCookie, attr) {
			t.Fatalf("cookie %q lacks %q", setCookie, attr)
		}
	}
	if admin.session == "" {
		t.Fatal("no session after setup")
	}
	if res := admin.expect("GET", "/api/setup/status", "", 200); res.body["needsSetup"] != false {
		t.Fatalf("status body %s", res.raw)
	}

	// Second setup is rejected, even with invalid input.
	other := &client{t: t, h: h}
	if res := other.expect("POST", "/api/setup", `{"username":"evil","password":"password1"}`, 409); res.body["error"] != "setup already completed" {
		t.Fatalf("409 body %s", res.raw)
	}
	other.expect("POST", "/api/setup", `{"username":"x","password":"short"}`, 409)

	// me -> logout -> me 401.
	if res := admin.expect("GET", "/api/auth/me", "", 200); userField(t, res, "username") != "admin" {
		t.Fatalf("me body %s", res.raw)
	}
	stale := admin.session
	admin.expect("POST", "/api/auth/logout", `{}`, 204)
	if admin.session != "" {
		t.Fatal("cookie not cleared on logout")
	}
	admin.expect("GET", "/api/auth/me", "", 401)
	admin.session = stale // server-side session must be gone too
	admin.expect("GET", "/api/auth/me", "", 401)
	admin.session = ""

	// Login (case-insensitive username).
	res = admin.expect("POST", "/api/auth/login", `{"username":"ADMIN","password":"password1"}`, 200)
	if userField(t, res, "username") != "admin" || admin.session == "" {
		t.Fatalf("login body %s", res.raw)
	}

	// Logging in again replaces the presented session server-side.
	first := admin.session
	admin.expect("POST", "/api/auth/login", `{"username":"admin","password":"password1"}`, 200)
	if admin.session == first {
		t.Fatal("session not rotated")
	}
	second := admin.session
	admin.session = first
	admin.expect("GET", "/api/auth/me", "", 401)
	admin.session = second
	admin.expect("GET", "/api/auth/me", "", 200)

	// Admin creates a user; duplicate is 409; bad role is 400.
	res = admin.expect("POST", "/api/users", `{"username":"Bob","password":"password2"}`, 201)
	if userField(t, res, "username") != "bob" || userField(t, res, "role") != "user" {
		t.Fatalf("create body %s", res.raw)
	}
	admin.expect("POST", "/api/users", `{"username":"bob","password":"password2"}`, 409)
	admin.expect("POST", "/api/users", `{"username":"carol","password":"password2","role":"root"}`, 400)
	admin.expect("POST", "/api/users", `{"username":"carol","password":"password2","role":""}`, 400)
	res = admin.expect("GET", "/api/users", "", 200)
	if users, _ := res.body["users"].([]any); len(users) != 2 {
		t.Fatalf("users body %s", res.raw)
	}

	// Non-admin is forbidden from /api/users.
	bob := &client{t: t, h: h}
	bob.expect("GET", "/api/users", "", 401)
	bob.expect("POST", "/api/auth/login", `{"username":"bob","password":"password2"}`, 200)
	bob.expect("GET", "/api/users", "", 403)
	bob.expect("POST", "/api/users", `{"username":"mallory","password":"password3"}`, 403)

	// 415 without JSON content type.
	for _, ct := range []map[string]string{{}, {"Content-Type": "text/plain"}, {"Content-Type": "application/x-www-form-urlencoded"}} {
		if res := bob.do("POST", "/api/auth/logout", `{}`, ct); res.status != 415 {
			t.Fatalf("content type %v: status %d", ct, res.status)
		}
	}
	if res := bob.do("POST", "/api/auth/logout", `{}`, map[string]string{"Content-Type": "application/json; charset=utf-8"}); res.status != 204 {
		t.Fatalf("json with charset: status %d", res.status)
	}

	// Wrong password and unknown user share the same 401 message.
	anon := &client{t: t, h: h}
	wrong := anon.expect("POST", "/api/auth/login", `{"username":"bob","password":"wrongpass"}`, 401)
	unknown := anon.expect("POST", "/api/auth/login", `{"username":"nobody","password":"wrongpass"}`, 401)
	if wrong.body["error"] != "invalid username or password" || unknown.body["error"] != wrong.body["error"] {
		t.Fatalf("401 bodies %s / %s", wrong.raw, unknown.raw)
	}
	anon.expect("POST", "/api/auth/login", `not json`, 400)

	// Rate limit: 10 failures per IP, then 429 even for correct credentials.
	for range 8 {
		anon.expect("POST", "/api/auth/login", `{"username":"bob","password":"wrongpass"}`, 401)
	}
	res = anon.expect("POST", "/api/auth/login", `{"username":"bob","password":"password2"}`, 429)
	if res.header.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}

func TestRouting(t *testing.T) {
	c := &client{t: t, h: newHandler(t)}
	tests := []struct {
		method, path string
		want         int
		wantBody     string
	}{
		{"GET", "/api/nope", 404, `{"error":"not found"}`},
		{"GET", "/api", 404, `{"error":"not found"}`},
		{"PUT", "/api/health", 405, `{"error":"method not allowed"}`},
		{"DELETE", "/api/users", 405, `{"error":"method not allowed"}`},
		{"GET", "/api/auth/login", 405, `{"error":"method not allowed"}`},
		{"GET", "/", 200, "spa"},
		{"GET", "/some/client/route", 200, "spa"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			res := c.do(tt.method, tt.path, "", nil)
			if res.status != tt.want || strings.TrimSpace(res.raw) != tt.wantBody {
				t.Fatalf("got %d %q", res.status, res.raw)
			}
		})
	}
	for path, allow := range map[string]string{
		"/api/health": "GET, HEAD", "/api/users": "GET, HEAD, POST", "/api/auth/login": "POST",
	} {
		if res := c.do("PATCH", path, "", nil); res.header.Get("Allow") != allow {
			t.Fatalf("PATCH %s: Allow = %q, want %q", path, res.header.Get("Allow"), allow)
		}
	}
	// 405 takes precedence over the Content-Type check.
	if res := c.do("PUT", "/api/health", "x", map[string]string{"Content-Type": "text/plain"}); res.status != 405 {
		t.Fatalf("PUT text/plain: status %d", res.status)
	}

	const apiCSP = "default-src 'none'; frame-ancestors 'none'"
	for _, p := range []string{"/api/health", "/api", "/api/nope"} {
		res := c.do("GET", p, "", nil)
		if res.header.Get("Cache-Control") != "no-store" || res.header.Get("X-Content-Type-Options") != "nosniff" ||
			res.header.Get("Content-Security-Policy") != apiCSP {
			t.Fatalf("%s headers %v", p, res.header)
		}
	}
}

func TestBodyLimit(t *testing.T) {
	c := &client{t: t, h: newHandler(t)}
	big := `{"username":"` + strings.Repeat("a", 1<<20) + `","password":"password1"}`
	c.expect("POST", "/api/setup", big, 413)
	c.expect("POST", "/api/setup", ``, 400)
	c.expect("POST", "/api/setup", `{"username":"admin","password":"password1"} {}`, 400)
	c.expect("POST", "/api/setup", `{"username":1,"password":"password1"}`, 400)
}
