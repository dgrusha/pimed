package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"runtime/debug"
	"time"

	"pimed/internal/app"
	"pimed/internal/domain/user"
)

type ctxKey struct{}

func withUser(ctx context.Context, u user.User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

func userFrom(ctx context.Context) (user.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(user.User)
	return u, ok
}

// authenticated resolves the session cookie to a user and stores it in the
// request context; otherwise it answers 401.
func (a *api) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := sessionToken(r)
		u, err := a.svc.CurrentUser(r.Context(), token)
		if errors.Is(err, app.ErrUnauthenticated) && token != "" {
			a.cookies.clear(w) // drop stale cookie
		}
		if err != nil {
			a.respondError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), u)))
	})
}

// requireAdmin answers 403 unless the authenticated user is an admin. The
// use cases enforce this too; this is defence in depth.
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, ok := userFrom(r.Context()); !ok || !u.IsAdmin() {
			writeError(w, http.StatusForbidden, "admin privileges required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireJSON rejects non-GET/HEAD/OPTIONS requests whose Content-Type is not
// application/json (415). Together with SameSite=Strict this is the CSRF
// defence: cross-site forms cannot send that content type.
func requireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mt != "application/json" {
				writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// apiCSP is the Content-Security-Policy for API responses: they are data,
// never documents, so nothing may load or frame them.
const apiCSP = "default-src 'none'; frame-ancestors 'none'"

// apiHeaders prevents caching of API responses and sets their CSP.
func apiHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("Content-Security-Policy", apiCSP)
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the response status for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += int64(n)
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// logRequests logs one line per request. Query strings, bodies and cookies
// are never logged.
func logRequests(log *slog.Logger, ips ipResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		level := slog.LevelInfo
		if rec.status >= 500 {
			level = slog.LevelError
		}
		log.Log(r.Context(), level, "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration", time.Since(start).Round(time.Microsecond),
			"ip", ips.clientIP(r),
		)
	})
}

// recoverPanics turns handler panics into 500 responses.
func recoverPanics(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			log.ErrorContext(r.Context(), "panic serving request",
				"method", r.Method, "path", r.URL.Path, "panic", v, "stack", string(debug.Stack()))
			writeError(w, http.StatusInternalServerError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}
