package httpapi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"

	httpapi "pimed/internal/adapters/inbound/http"
	"pimed/internal/app"
	"pimed/internal/domain/user"
)

// login posts credentials from remoteAddr (with optional X-Forwarded-For)
// and returns the status code.
func login(h http.Handler, remoteAddr, xff, username, password string) *httptest.ResponseRecorder {
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func setupAdmin(t *testing.T, h http.Handler) {
	t.Helper()
	c := &client{t: t, h: h}
	c.expect("POST", "/api/setup", `{"username":"admin","password":"password1"}`, 201)
}

func TestLoginBurstIsLimited(t *testing.T) {
	h := newHandler(t)
	setupAdmin(t, h)

	const n = 30
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		counts = map[int]int{}
	)
	start := make(chan struct{})
	for range n {
		wg.Go(func() {
			<-start
			code := login(h, "198.51.100.7:4000", "", "admin", "wrong-password").Code
			mu.Lock()
			counts[code]++
			mu.Unlock()
		})
	}
	close(start)
	wg.Wait()
	if counts[401] != httpapi.DefaultLoginMaxFailures || counts[429] != n-httpapi.DefaultLoginMaxFailures {
		t.Fatalf("status counts %v", counts)
	}
}

func TestLoginSuccessDoesNotResetIP(t *testing.T) {
	h := newHandler(t)
	setupAdmin(t, h)
	const ip = "198.51.100.8:4000"

	// An attacker burns 9 attempts on assorted usernames...
	for i := range 9 {
		if code := login(h, ip, "", fmt.Sprintf("ghost%d", i), "wrong-password").Code; code != 401 {
			t.Fatalf("attempt %d: status %d", i, code)
		}
	}
	// ...a legitimate user behind the same IP signs in...
	if code := login(h, ip, "", "admin", "password1").Code; code != 200 {
		t.Fatalf("legit login: status %d", code)
	}
	// ...which must not give the attacker a fresh budget.
	if code := login(h, ip, "", "ghost9", "wrong-password").Code; code != 401 {
		t.Fatalf("10th failure: status %d", code)
	}
	if code := login(h, ip, "", "ghost10", "wrong-password").Code; code != 429 {
		t.Fatalf("after 10 failures: status %d", code)
	}
}

func TestLoginPerUsernameLockout(t *testing.T) {
	h := newHandler(t)
	setupAdmin(t, h)
	for i := range 10 {
		addr := fmt.Sprintf("203.0.113.%d:4000", i+1)
		if code := login(h, addr, "", "ADMIN", "wrong-password").Code; code != 401 {
			t.Fatalf("attempt %d: status %d", i, code)
		}
	}
	rec := login(h, "203.0.113.200:4000", "", "admin", "password1")
	if rec.Code != 429 || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("locked username from new IP: status %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	// Other usernames from the same fresh IP are unaffected.
	if code := login(h, "203.0.113.200:4000", "", "someone", "wrong-password").Code; code != 401 {
		t.Fatalf("other username: status %d", code)
	}
}

func TestLoginBehindTrustedProxy(t *testing.T) {
	h := newHandlerWith(t, httpapi.Options{
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")},
	})
	setupAdmin(t, h)
	// Client A behind the proxy exhausts its budget (distinct usernames, so
	// only the IP key is involved).
	for i := range 10 {
		login(h, "127.0.0.1:5000", "192.0.2.10", fmt.Sprintf("user%d", i), "wrong-password")
	}
	if code := login(h, "127.0.0.1:5000", "192.0.2.10", "other", "wrong-password").Code; code != 429 {
		t.Fatalf("client A: status %d", code)
	}
	// Client B behind the same proxy is a different key.
	if code := login(h, "127.0.0.1:5000", "192.0.2.11", "other", "wrong-password").Code; code != 401 {
		t.Fatalf("client B: status %d", code)
	}
	// Client A cannot escape by spoofing an extra XFF entry: the proxy appends
	// the real address on the right.
	if code := login(h, "127.0.0.1:5000", "10.9.9.9, 192.0.2.10", "other2", "wrong-password").Code; code != 429 {
		t.Fatalf("spoofing client A: status %d", code)
	}
}

// busyService fails every login with user.ErrHasherBusy.
type busyService struct{ httpapi.Service }

func (busyService) Login(context.Context, string, string) (app.AuthResult, error) {
	return app.AuthResult{}, fmt.Errorf("verify password: %w", user.ErrHasherBusy)
}

func TestLoginBusy(t *testing.T) {
	h := httpapi.NewHandler(busyService{}, httpapi.Options{})
	// Busy answers are 503 and do not consume the rate-limit budget.
	for i := range 2 * httpapi.DefaultLoginMaxFailures {
		rec := login(h, "192.0.2.50:1", "", "admin", "password1")
		if rec.Code != 503 || strings.TrimSpace(rec.Body.String()) != `{"error":"server busy, try again"}` {
			t.Fatalf("attempt %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
}
