// Package config loads PiMed's runtime configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// DefaultTrustedProxies is used when PIMED_TRUSTED_PROXIES is unset: Caddy
// running on the same host.
const DefaultTrustedProxies = "127.0.0.1/32,::1/128"

// Config is the validated runtime configuration (see docs/api.md).
type Config struct {
	Addr         string
	DataDir      string
	DBPath       string
	CookieSecure bool
	SessionTTL   time.Duration
	LogLevel     slog.Level
	// TrustedProxies are the reverse proxies whose X-Forwarded-For header is
	// believed. Empty means the TCP peer is always the client.
	TrustedProxies []netip.Prefix
}

// Load reads configuration via lookup (normally os.LookupEnv), applying
// defaults and validating values. All problems are reported together.
func Load(lookup func(string) (string, bool)) (Config, error) {
	get := func(key, def string) string {
		if v, _ := lookup(key); strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		return def
	}
	cfg := Config{
		Addr:    get("PIMED_ADDR", "127.0.0.1:8080"),
		DataDir: get("PIMED_DATA_DIR", "/data"),
		DBPath:  get("PIMED_DB_PATH", "/var/lib/pimed/pimed.db"),
	}
	var errs []error

	secure, err := strconv.ParseBool(get("PIMED_COOKIE_SECURE", "true"))
	if err != nil {
		errs = append(errs, errors.New("PIMED_COOKIE_SECURE: must be true or false"))
	}
	cfg.CookieSecure = secure

	ttl, err := time.ParseDuration(get("PIMED_SESSION_TTL", "720h"))
	switch {
	case err != nil:
		errs = append(errs, fmt.Errorf("PIMED_SESSION_TTL: %w", err))
	case ttl < time.Minute:
		errs = append(errs, errors.New("PIMED_SESSION_TTL: must be at least 1m"))
	}
	cfg.SessionTTL = ttl

	switch lvl := strings.ToLower(get("PIMED_LOG_LEVEL", "info")); lvl {
	case "debug":
		cfg.LogLevel = slog.LevelDebug
	case "info":
		cfg.LogLevel = slog.LevelInfo
	case "warn":
		cfg.LogLevel = slog.LevelWarn
	case "error":
		cfg.LogLevel = slog.LevelError
	default:
		errs = append(errs, fmt.Errorf("PIMED_LOG_LEVEL: unknown level %q (want debug, info, warn or error)", lvl))
	}

	// Unset means the default; set but empty means "trust no proxy".
	proxies, ok := lookup("PIMED_TRUSTED_PROXIES")
	if !ok {
		proxies = DefaultTrustedProxies
	}
	cfg.TrustedProxies, err = ParsePrefixes(proxies)
	if err != nil {
		errs = append(errs, fmt.Errorf("PIMED_TRUSTED_PROXIES: %w", err))
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

// ParsePrefixes parses a comma-separated list of CIDRs and/or bare IP
// addresses (a bare address becomes a single-host prefix). Empty items are
// ignored, so "" yields an empty list.
func ParsePrefixes(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for item := range strings.SplitSeq(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			p, err := netip.ParsePrefix(item)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q", item)
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("invalid IP address %q", item)
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}
