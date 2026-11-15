// Package server runs the HTTP server with graceful shutdown.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// ShutdownTimeout bounds how long in-flight requests may take to finish.
const ShutdownTimeout = 15 * time.Second

// New returns an http.Server with conservative timeouts.
//
// No ReadTimeout/WriteTimeout is set on purpose: future upload and media
// streaming handlers need long-lived bodies. Handlers that need deadlines
// should set them per request via http.ResponseController.
func New(addr string, h http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}

// Run listens on srv.Addr and serves until ctx is cancelled, then shuts down
// gracefully.
func Run(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("server: listen %s: %w", srv.Addr, err)
	}
	log.Info("http server listening", "addr", ln.Addr().String())

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return fmt.Errorf("server: serve: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down http server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server: shutdown: %w", err)
	}
	if err := <-errc; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server: serve: %w", err)
	}
	return nil
}
