// Command pimed is the PiMed server. This file is the composition root: it
// loads configuration and wires adapters, use cases and handlers together.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	httpapi "pimed/internal/adapters/inbound/http"
	"pimed/internal/adapters/outbound/argon2id"
	"pimed/internal/adapters/outbound/sqlite"
	"pimed/internal/adapters/outbound/token"
	"pimed/internal/app"
	"pimed/internal/platform/config"
	"pimed/internal/platform/logging"
	"pimed/internal/platform/server"
	"pimed/internal/platform/spa"
	"pimed/web"
)

const sessionCleanupInterval = time.Hour

// version is set at build time by release builds:
// go build -ldflags "-X main.version=v1.2.3" ./cmd/pimed
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println("pimed", version)
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "pimed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}
	log := logging.New(os.Stderr, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o750); err != nil {
		return fmt.Errorf("create database dir: %w", err)
	}

	db, err := sqlite.Open(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	svc, err := app.New(app.Deps{
		Users:      sqlite.NewUserRepository(db),
		Sessions:   sqlite.NewSessionRepository(db),
		Hasher:     argon2id.New(argon2id.DefaultParams()),
		Tokens:     token.Generator{},
		SessionTTL: cfg.SessionTTL,
	})
	if err != nil {
		return err
	}

	dist, err := web.Dist()
	if err != nil {
		return err
	}
	handler := httpapi.NewHandler(svc, httpapi.Options{
		Logger:         log,
		CookieSecure:   cfg.CookieSecure,
		SPA:            spa.Handler(dist),
		TrustedProxies: cfg.TrustedProxies,
	})

	// Background work must finish before the deferred db.Close runs.
	var wg sync.WaitGroup
	defer wg.Wait()
	defer stop()
	wg.Go(func() { cleanupSessions(ctx, svc, log) })

	log.Info("starting pimed", "version", version,
		"addr", cfg.Addr, "dataDir", cfg.DataDir, "dbPath", cfg.DBPath,
		"cookieSecure", cfg.CookieSecure, "sessionTTL", cfg.SessionTTL,
		"trustedProxies", cfg.TrustedProxies)
	return server.Run(ctx, server.New(cfg.Addr, handler, log), log)
}

// cleanupSessions prunes expired sessions periodically until ctx is done.
func cleanupSessions(ctx context.Context, svc *app.Service, log *slog.Logger) {
	ticker := time.NewTicker(sessionCleanupInterval)
	defer ticker.Stop()
	for {
		n, err := svc.PruneSessions(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			log.Warn("session cleanup failed", "err", err)
		case n > 0:
			log.Debug("expired sessions pruned", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
