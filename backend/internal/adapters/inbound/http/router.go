// Package httpapi is the inbound HTTP adapter: routing, handlers, DTOs and
// middleware for the JSON API described in docs/api.md. (The directory is
// named http; the package is httpapi to avoid shadowing net/http.)
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"pimed/internal/app"
	"pimed/internal/domain/user"
)

// Service is the subset of the application layer used by the handlers.
type Service interface {
	NeedsSetup(ctx context.Context) (bool, error)
	Setup(ctx context.Context, username, password string) (app.AuthResult, error)
	Login(ctx context.Context, username, password string) (app.AuthResult, error)
	Logout(ctx context.Context, token string) error
	CurrentUser(ctx context.Context, token string) (user.User, error)
	CreateUser(ctx context.Context, actor user.User, in app.CreateUserInput) (user.User, error)
	ListUsers(ctx context.Context, actor user.User) ([]user.User, error)
}

// Options configure the HTTP adapter.
type Options struct {
	Logger       *slog.Logger
	CookieSecure bool
	Limiter      *LoginLimiter // nil means DefaultLoginLimiter()
	SPA          http.Handler  // serves every non-/api path; nil means 404
	// TrustedProxies are the reverse proxies whose X-Forwarded-For header is
	// believed when determining the client IP. Empty trusts no one.
	TrustedProxies []netip.Prefix
}

type api struct {
	svc     Service
	log     *slog.Logger
	cookies cookieConfig
	limiter *LoginLimiter
	ips     ipResolver
}

// NewHandler builds the complete HTTP handler: API routes, SPA fallback and
// the middleware chain.
func NewHandler(svc Service, opts Options) http.Handler {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.DiscardHandler)
	}
	if opts.Limiter == nil {
		opts.Limiter = DefaultLoginLimiter()
	}
	if opts.SPA == nil {
		opts.SPA = http.NotFoundHandler()
	}
	a := &api{
		svc:     svc,
		log:     opts.Logger,
		cookies: cookieConfig{secure: opts.CookieSecure},
		limiter: opts.Limiter,
		ips:     ipResolver{trusted: slices.Clone(opts.TrustedProxies)},
	}

	// JSON endpoints get a body read deadline (withJSONReadDeadline). Future
	// streaming/upload routes must be registered without it.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", a.health)
	mux.HandleFunc("GET /api/setup/status", a.setupStatus)
	mux.Handle("POST /api/setup", withJSONReadDeadline(http.HandlerFunc(a.setup)))
	mux.Handle("POST /api/auth/login", withJSONReadDeadline(http.HandlerFunc(a.login)))
	mux.Handle("POST /api/auth/logout", withJSONReadDeadline(http.HandlerFunc(a.logout)))
	mux.Handle("GET /api/auth/me", a.authenticated(http.HandlerFunc(a.me)))
	mux.Handle("GET /api/users", a.authenticated(requireAdmin(http.HandlerFunc(a.listUsers))))
	mux.Handle("POST /api/users", withJSONReadDeadline(a.authenticated(requireAdmin(http.HandlerFunc(a.createUser)))))

	root := http.NewServeMux()
	root.Handle("/api/", apiHeaders(jsonFallbacks(mux, requireJSON(mux))))
	root.Handle("/api", apiHeaders(http.HandlerFunc(notFound)))
	root.Handle("/", opts.SPA)

	return recoverPanics(opts.Logger, logRequests(opts.Logger, a.ips, securityHeaders(root)))
}

// probeMethods are tried to build the Allow header for a 405.
var probeMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
}

// jsonFallbacks passes requests that match a route of mux on to next (which
// ends in mux), but answers unmatched requests itself with JSON errors: 405
// (plus Allow) when the path exists for other methods, otherwise 404.
func jsonFallbacks(mux *http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			next.ServeHTTP(w, r)
			return
		}
		var allow []string
		for _, m := range probeMethods {
			probe := r.WithContext(r.Context()) // shallow copy
			probe.Method = m
			if _, pattern := mux.Handler(probe); pattern != "" {
				allow = append(allow, m)
			}
		}
		if len(allow) == 0 {
			notFound(w, r)
			return
		}
		w.Header().Set("Allow", strings.Join(allow, ", "))
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

func (a *api) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
